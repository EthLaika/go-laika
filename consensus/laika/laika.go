// Package laika implements the laika proof-of-capacity algorithm
package laika

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/crypto/sha3"
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errInvalidPoC is thrown if the verification of the PoC fails
	errInvalidPoC = errors.New("invalid poc")
	// errZeroBlockTime is thrown if the blocks timestamp equals the timestamp of the parent
	errZeroBlockTime = errors.New("timestamp equals parent's")
	// errInvalidDifficulty is thrown if someone provides a difficulty < 0
	errInvalidDifficulty = errors.New("invalid difficulty")
	// errHigherDifficultyFound is thrown if we received a block with higher difficulty until blocktime
	errHigherDifficultyFound = errors.New("higher difficulty found")

	errQuit = errors.New("received quit signal")
)

var (
	// Max time from current time allowed for blocks, before they're considered future blocks
	allowedFutureBlockTime = 15 * time.Second
	// Block reward given every block
	blockReward = big.NewInt(2e+18)
)

type bestResult struct {
	difficulty *big.Int      // achieved difficulty of the best result
	header     *types.Header // pointer to the best result
}

// Laika is the proof-of-capacity consensus engine for the Ethereum Laika testnet.
type Laika struct {
	config      *params.LaikaConfig      // Consensus engine configuration parameters
	db          ethdb.Database           // Database to store and retrieve snapshot checkpoints
	miningLock  sync.Mutex               // Ensures thread safety for the in-memory caches and mining fields
	update      chan struct{}            // Notification channel to update mining parameters
	hashrate    metrics.Meter            // Meter tracking the average hashrate
	wg          sync.WaitGroup           // Waitgroup of verification threads waiting for the next blocktime
	result      bestResult               // Best found result for this block period
	resultGuard sync.RWMutex             // Guard protecting the access to the best Result
	file        *PlotFile                // The plot file used for mining.
	newHeader   chan *types.Header       // communicates to the metronome a new header on which mining has started
	veriSync    map[uint64]chan struct{} // synchronizes calls to VerifyHeader per block number
	quit        chan struct{}            // quits running routines like metronome
}

// New creates a Laika proof-of-capacity consensus engine
func New(config *params.LaikaConfig, datasetDir string, db ethdb.Database) *Laika {
	if datasetDir != "" {
		log.Info("Disk storage enabled for ethash DAGs", "dir", datasetDir)
	}
	file := OpenPlotFile(datasetDir)
	if file == nil {
		panic("could not open the plot file '" + datasetDir + "'")
	}

	engine := &Laika{
		config:    config,
		db:        db,
		file:      file,
		newHeader: make(chan *types.Header, 1),
		veriSync:  make(map[uint64]chan struct{}),
		quit:      make(chan struct{}),
	}
	go engine.metronome()

	return engine
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (l *Laika) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (l *Laika) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return l.verifyHeaderWorker(chain, []*types.Header{header}, []bool{false}, 0)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (l *Laika) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = l.verifyHeaderWorker(chain, headers, seals, index)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (l *Laika) verifyHeaderWorker(chain consensus.ChainReader, headers []*types.Header, seals []bool, index int) error {
	l.wg.Add(1)
	defer l.wg.Done()

	var parent *types.Header
	header := headers[index]
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == header.ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	if chain.GetHeader(header.Hash(), header.Number.Uint64()) != nil {
		return nil // known block
	}

	if l.verifyHeader(chain, header, parent, false, seals[index]) != nil {
		return errInvalidPoC
	}

	// header passed the verification, we'll check it against all other current
	// checks and wait
	difficulty := new(big.Int).SetBytes(headerHash(header))

	l.miningLock.Lock()
	if l.result.difficulty == nil || difficulty.Cmp(l.result.difficulty) == -1 {
		l.result.difficulty = difficulty
		l.result.header = header
	}
	l.miningLock.Unlock()

	blockNum := header.Number.Uint64()
	log.Debug("[verifyHeaderWorker] waiting for tick", "blockNum", blockNum)
	select {
	case <-l.veriSync[blockNum]:
	case <-l.quit:
		return errQuit
	}

	// A better result was received in the meantime
	if l.result.header.Hash() != header.Hash() {
		return errHigherDifficultyFound
	}

	// reset the best result
	l.miningLock.Lock()
	l.result.difficulty = nil
	l.result.header = nil
	l.miningLock.Unlock()
	return nil
}

// verifyHeader checks whether a header conforms to the consensus rules
func (l *Laika) verifyHeader(chain consensus.ChainReader, header, parent *types.Header, uncle bool, seal bool) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// Verify the header's timestamp
	if !uncle {
		if header.Time > uint64(time.Now().Add(allowedFutureBlockTime).Unix()) {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time <= parent.Time {
		return errZeroBlockTime
	}

	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := int64(parent.GasLimit) - int64(header.GasLimit)
	if diff < 0 {
		diff *= -1
	}
	limit := parent.GasLimit / params.GasLimitBoundDivisor

	if uint64(diff) >= limit || header.GasLimit < params.MinGasLimit {
		return fmt.Errorf("invalid gas limit: have %d, want %d += %d", header.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Verify the engine specific seal securing the block
	if seal {
		if err := l.VerifySeal(chain, header); err != nil {
			return err
		}
	}
	if err := misc.VerifyForkHashes(chain.Config(), header, uncle); err != nil {
		return err
	}
	return nil
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism does not permit uncles.
func (l *Laika) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// VerifySeal implements consensus.Engine, checking whether the given block satisfies
// the PoW difficulty requirements.
func (l *Laika) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	// Ensure that we have a valid difficulty for the block
	if header.Difficulty.Sign() <= 0 {
		return errInvalidDifficulty
	}
	return VrfyProof(header)
}

// Prepare implements consensus.Engine, initializing the  parent.
func (l *Laika) Prepare(chain consensus.ChainReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	l.newHeader <- header
	return nil
}

// Finalize implements consensus.Engine, accumulating the block and uncle rewards,
// setting the final state on the header
func (l *Laika) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	// Accumulate any block and uncle rewards and commit the final state root
	state.AddBalance(header.Coinbase, blockReward)
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
}

// FinalizeAndAssemble implements consensus.Engine, accumulating the block and
// uncle rewards, setting the final state and assembling the block.
func (l *Laika) FinalizeAndAssemble(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// Accumulate any block and uncle rewards and commit the final state root
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	state.AddBalance(header.Coinbase, blockReward)
	// Header seems complete, assemble into a block and return
	return types.NewBlock(header, txs, uncles, receipts), nil
}

func (l *Laika) Seal(chain consensus.ChainReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	go func() {
		// TODO add ChanIterator for correct column
		header := block.Header()
		GenProof(header, l.file.Iterator(challengeCol(headerHash(header))))
		results <- block.WithSeal(header)
	}()

	return nil
}

// SealHash returns the hash of a block prior to it being sealed.
func (l *Laika) SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.New256()
	headerEncode(hasher, header)
	hasher.Sum(hash[:0])
	return hash
}

// CalcDifficulty is a noop fo the laika algorithm.
func (l *Laika) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return big.NewInt(1)
}

// APIs returns the RPC APIs this consensus engine provides.
func (l *Laika) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "laika",
		Version:   "1.0",
		Service:   &API{laika: l},
		Public:    false,
	}}
}

// Hashrate returns the hashrate of the current miner
func (l *Laika) Hashrate() uint64 {
	return 0xB0B0
}

// Close implements consensus.Engine.
func (l *Laika) Close() error {
	close(l.quit)
	close(l.newHeader) // Prepare() should not be called after a call to Close()
	return nil
}

// The metronome is a go routine that sends
func (l *Laika) metronome() {
	// we start a ticker on the next full multiple of a period in this minute
	tickerStart := nextFullMultiple(l.config.Period)
	log.Debug("[metronome] Starting ticker on", "ts", tickerStart)
	time.Sleep(time.Until(tickerStart))

	ticker := time.NewTicker(time.Duration(l.config.Period) * time.Second)
	defer ticker.Stop()

	for h := range l.newHeader {
		blockNum := h.Number.Uint64()
		log.Debug("[metronome] Received new header #", "blockNum", blockNum)
		l.veriSync[blockNum] = make(chan struct{})
		select {
		case <-ticker.C:
			log.Debug("[metronome] Tick for header #", "blockNum", blockNum)
		case <-l.quit:
			return
		}
		// signal to all verifyHeaderWorkers that proof generation period is done
		// for this block
		close(l.veriSync[blockNum])
		delete(l.veriSync, blockNum)
	}
}

func nextFullMultiple(period uint64) time.Time {
	now := time.Now()
	if period < 60 && period%60 == 0 {
		nowSec := now.Second()
		fullSec := (uint64(nowSec) + period - 1) / period * period
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), int(fullSec), 0, now.Location())
	} else if now.Second() == 0 {
		return now
	} else {
		return time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())
	}
}
