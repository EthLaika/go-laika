# Laika consensus

> A proof of capacity consensus protocol

In this project, we developed a consensus algorithm and implemented it in go-ethereum.
We set up an Ethereum test net running our consensus algorithm.

**Proof of Capacity**&emsp;
The proof of capacity approach used in this project was inspired by Burst's consensus algorithm.
Proof of capacaity is similar to proof of work in that it is a probabilistical consensus algorithm, but instead of generating mining power from a miner's hash rate, mining power is generated from disk capacity, which is more energy efficient.
Unlike proof of work, there is a fixed block time.
Before you can start to mine, you first have to prepare your disk using a process called *plotting*.
During plotting, you solve computationally intensive puzzles, and write the solutions to your hard drive.
When you finished plotting, you can start mining.
When mining, random subsets of the plotted solutions are chosen, depending on the block's hash.
The miner looks through all the chosen solutions, and finds the best solution for the current block, and then publishes it.
The best solution that is received within one block time is chosen as the new block.
As there is a fixed block time, you start mining at the beginning of the block time, until you finished looking through all your solutions.
You then stop mining and publish the best solution.
Until the next block begins, the miner is idle and only has to collect and relay transactions.
This idling period is what makes the proof of capacity schemes much more energy-efficient than proof of work schemes.
Even low-power devices such as Raspberry Pis could be used to mine with no disadvantage against more powerful machines.

**Laika consensus**&emsp;
In Laika consensus, an improvement of Burst's algorithm is used: We replaced the puzzle function with a function that is much more efficient to verify.
Burst's puzzle function cannot be verified faster than it is generated, so its verification is very expensive.
For our puzzle function, verification is about 1000 times faster than generation.
This prevents the miners from getting DoS'ed with fake challenges that take much time to verify.

**The Laika test net**&emsp;
We set up an Ethereum test net with geth nodes that are modified to use the Laika consensus.
The boot node can be found at [laika.cash](https://laika.cash/).

# TODO: setup

# TODO: run