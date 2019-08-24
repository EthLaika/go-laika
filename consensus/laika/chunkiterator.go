package laika

// ChunkIterator iterates over all chunks within one of a plot file's columns.
type ChunkIterator struct {
	ColumnBlock
	file   *PlotFile
	column int
	index  int
}

// Next retrieves the next chunk to be iterated, if it exists.
// Returns whether there is a next chunk.
// When the end of the file is reached, resets the file.
func (i *ChunkIterator) Next() bool {
	// Is the iterator's cached data exhausted?
	if i.index >= int(i.Length) {
		var err error
		if i.ColumnBlock, err = i.file.ReadOneBlock(i.column); err != nil {
			i.Destroy()
			return false
		} else {
			i.index = 1
			if i.Length != 0 {
				return true
			} else {
				i.Destroy()
				return false
			}
		}
	}

	i.index++
	return true
}

// Destroys a chunk iterator.
// Resets the file so that another iterator may be created.
func (i *ChunkIterator) Destroy() {
	i.file.mutex.Lock()
	defer i.file.mutex.Unlock()

	if !i.file.used {
		panic("iterator.Destroy: was already destroyed")
	}
	i.file.used = false

	i.file.reset()
}

// Chunk retrieves the last chunk retrieved via Next.
// Next always has to be called before Chunk.
func (i *ChunkIterator) Chunk() Chunk {
	if i.index == 0 || i.index > len(i.witnesses) {
		panic("illegal iterator access")
	}

	return Chunk{
		chunk: i.chunks[(i.index-1)*M : i.index*M],
		idx:   i.Start + uint64(i.index),
		nonce: i.witnesses[i.index],
	}
}
