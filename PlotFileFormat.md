# Plotfile format

The plotfile starts with the following header:

| Name | Type | Description |
:------:------:-------------:
| *a*  | Address ([20]byte) | The miner's ethereum address. |
| *M*  | uint8 | The number of bytes per chunk. |
| *N*  | uint16 (LE) | The number of chunks/columns per row. |
| *K*  | uint8 | The number of iterations used in G<sub>i,r</sub>. |
| *D*  | \[32\]byte | The difficulty used in this plot file. |

It is followed by blocks of plotted data, where each block looks as follows:

| Name | Type | Description |
:------:------:-------------:
| start | uint64 (LE) | The row index of the first row in this block. |
| length | uint16 (LE) | The number of rows in this block. |
| columns | \[*N*\]\[length\]\[*M*\]byte | The solution chunks, grouped by their column index. |
| witnesses | \[length\]uint32 | The witnesses that belong to the rows. |

In plotted blocks, all rows have consecutive row indices, beginning with `start`.
When mining, only the column that is selected by the block hash is read, the best solution is found, and then the matching witness is read.
Afterwards, the next block is read, until the end of the file is reached.