package main

import (
	"fmt"
	"io"
	"math"
)

type Faces struct {
	xPos, zPos int
	count      int

	vertexes  Vertexes
	texcoords TexCoords
	faces     []Face
	boundary  *BoundaryLocator
}

func (fs *Faces) ProcessChunk(enclosed *EnclosedChunk, w io.Writer) (count int) {
	fs.Clean(enclosed.xPos, enclosed.zPos)
	fs.processBlocks(enclosed, fs)
	fs.Write(w)
	return len(fs.faces)
}

func (fs *Faces) Clean(xPos, zPos int) {
	fs.xPos = xPos
	fs.zPos = zPos

	if fs.vertexes == nil {
		fs.vertexes = make([]int16, (128+1)*(16+1)*(16+1))
	} else {
		fs.vertexes.Clear()
	}

	if fs.faces == nil {
		fs.faces = make([]Face, 0, 8192)
	} else {
		fs.faces = fs.faces[:0]
	}
}

type AddFacer interface {
	AddFace(blockId uint16, v1, v2, v3, v4 Vertex)
}

type Face struct {
	blockId    uint16
	indexes    [4]int
	texIndexes [4]int
}

func (fs *Faces) AddFace(blockId uint16, v1, v2, v3, v4 Vertex) {

	//	fmt.Printf("mcobj %v\n", blockTypeMap[uint8(blockId&255)])
	var tc TexCoord = NullTexCoord() //blockTypeMap[uint8(blockId&255)]
	var face = Face{blockId, [4]int{fs.vertexes.Use(v1), fs.vertexes.Use(v2), fs.vertexes.Use(v3), fs.vertexes.Use(v4)}, [4]int{fs.texcoords.Use(tc, false, false, 0), fs.texcoords.Use(tc, true, false, 0), fs.texcoords.Use(tc, true, true, 0), fs.texcoords.Use(tc, false, true, 0)}}
	fs.faces = append(fs.faces, face)
}

func (fs *Faces) Write(w io.Writer) {
	fs.vertexes.Number()
	var vc = int16(fs.vertexes.Print(w, fs.xPos, fs.zPos))

	var blockIds = make([]uint16, 0, 16)
	for _, face := range fs.faces {
		var found = false
		for _, id := range blockIds {
			if id == face.blockId {
				found = true
				break
			}
		}

		if !found {
			blockIds = append(blockIds, face.blockId)
		}
	}

	for _, blockId := range blockIds {
		printMtl(w, blockId)
		for _, face := range fs.faces {
			if face.blockId == blockId {
				fmt.Fprintln(w, "f", fs.vertexes.Get(face.indexes[0])-vc-1, fs.vertexes.Get(face.indexes[1])-vc-1, fs.vertexes.Get(face.indexes[2])-vc-1, fs.vertexes.Get(face.indexes[3])-vc-1)
				faceCount++
			}
		}
	}
}

type Vertexes []int16

func (vs *Vertexes) Index(x, y, z int) int {
	return y + (z*129 + (x * 129 * 17))
}

func (vs *Vertexes) Use(v Vertex) int {
	var i = vs.Index(v.x, v.y, v.z)
	(*vs)[i]++
	return i
}

func (vs *Vertexes) Release(v Vertex) int {
	var i = vs.Index(v.x, v.y, v.z)
	(*vs)[i]--
	return i
}

func (vs *Vertexes) Get(i int) int16 {
	return (*vs)[i]
}

func (vs *Vertexes) Clear() {
	for i, _ := range *vs {
		(*vs)[i] = 0
	}
}

func (vs *Vertexes) Number() {
	var count int16 = 0
	for i, references := range *vs {
		if references != 0 {
			count++
			(*vs)[i] = count
		} else {
			(*vs)[i] = -1
		}
	}
}


const numBlockPatternsAcross = 16
const numBlockPatterns = numBlockPatternsAcross * numBlockPatternsAcross
const numNonrepeatingTexcoordsAcross = numBlockPatternsAcross * 2
const numNonrepeatingTexcoords = numBlockPatternsAcross * numBlockPatternsAcross

const numRepeatingPatternsAcross = 64
const numRepeatingTexcoordsAcross = numRepeatingPatternsAcross * 2
const maxDepth = 129 //fencepost (need both top and bottom);

const totalNumTexcoords = numNonrepeatingTexcoords + maxDepth*numRepeatingTexcoordsAcross

type TexCoords [totalNumTexcoords]int16

func (vs *TexCoords) Index(x int, y int, xright bool, ybot bool, reps bool) int {
	var xoffset = 0
	if xright {
		xoffset = -1 //get the coordinate just on the closer edge of the pixel
	}
	var yoffset = 0
	if ybot {
		yoffset = -1 //get the coordinate just on the closer edge of the pixel
	}
	if x == 0 && xoffset < 0 {
		return 0
	}

	if reps {
		return x*2 + xoffset + numRepeatingTexcoordsAcross*y
	}
	if y == 0 && yoffset < 0 {
		return 0
	}
	return x*2 + xoffset + numNonrepeatingTexcoordsAcross*(y*2+yoffset)
}

func (tcs *TexCoords) Use(tc TexCoord, xright bool, ybot bool, numReps int) int {
	var i = 0
	if xright {
		if ybot {
			i = tcs.Index(int(tc.bottomRight.x), int(tc.bottomRight.y), xright, ybot, false) //fixme reps
		} else {
			i = tcs.Index(int(tc.bottomRight.x), int(tc.topLeft.y), xright, ybot, false) //fixme reps
		}
	} else {
		if ybot {
			i = tcs.Index(int(tc.topLeft.x), int(tc.bottomRight.y), xright, ybot, false) //fixme reps
		} else {
			i = tcs.Index(int(tc.topLeft.x), int(tc.topLeft.y), xright, ybot, false) //fixme reps
		}
	}
	(*tcs)[i]++
	return i
}

func (tcs *TexCoords) Release(tc TexCoord, xright bool, ybot bool, numReps int) int {
	var i = tcs.Use(tc, xright, ybot, numReps)
	(*tcs)[i]--
	(*tcs)[i]--
	return i
}

func (tcs *TexCoords) Get(i int) int16 {
	return (*tcs)[i]
}

func (tcs *TexCoords) Clear() {
	for i, _ := range *tcs {
		(*tcs)[i] = 0
	}
}

func (tcs *TexCoords) Number() {
	var count int16 = 0
	for i, references := range *tcs {
		if references != 0 {
			count++
			(*tcs)[i] = count
		} else {
			(*tcs)[i] = -1
		}
	}
}

func (tcs *TexCoords) Print(w io.Writer, imageWidth int, imageHeight int) (count int) {
	var buf = make([]byte, 64)
	copy(buf[0:3], "vt ")
	patternWidth := (imageWidth / numBlockPatternsAcross)
	count = 0
	for i := 0; i < numBlockPatternsAcross; i++ {
		for j := 0; j < numBlockPatternsAcross; j++ {
			for isub := 0; isub < 2; isub++ {
				for jsub := 0; jsub < 2; jsub++ {
					xPixel := i*patternWidth + isub*(patternWidth-1)
					yPixel := j*patternWidth + jsub*(patternWidth-1)
					index := i*2 + isub + (j*2+jsub)*numBlockPatternsAcross*2
					if (*tcs)[index] != -1 {
						xCoord := float64(xPixel) / float64(imageWidth-1)
						yCoord := float64(yPixel) / float64(imageHeight-1)
						buf = buf[:3]
						if xCoord == xCoord {
							buf = appendFloat(buf, xCoord)
						}
						buf = append(buf, ' ')
						if yCoord == yCoord {
							buf = appendFloat(buf, yCoord)
						}
						buf = append(buf, '\n')
					}
				}
			}
		}
	}

	for i := 0; i < numRepeatingPatternsAcross; i++ {
		for j := 0; j < maxDepth; j++ {
			for isub := 0; isub < 2; isub++ {
				xPixel := i*patternWidth + isub*(patternWidth-1)
				yPixel := j*patternWidth + (patternWidth - 1)
				if j == 0 {
					yPixel = 0
				}
				index := i*2 + isub + j*numRepeatingPatternsAcross*2 + numNonrepeatingTexcoords
				if (*tcs)[index] != -1 {
					xCoord := float64(xPixel) / float64(imageWidth-1)
					yCoord := float64(yPixel) / float64(imageHeight-1)
					buf = buf[:3]
					if xCoord == xCoord {
						buf = append(buf, '1') //appendCoord(buf, xCoord)
					}
					buf = append(buf, ' ')
					if yCoord == yCoord {
						buf = append(buf, '1') //appendCoord(buf, yCoord)
					}
					buf = append(buf, '\n')
				}
			}
		}
	}

	return
}

func (vs *Vertexes) Print(w io.Writer, xPos, zPos int) (count int) {
	var buf = make([]byte, 64)
	copy(buf[0:2], "v ")

	count = 0
	for i := 0; i < len(*vs); i += 129 {
		var x, z = (i / 129) / 17, (i / 129) % 17

		var column = (*vs)[i : i+129]
		for y, offset := range column {
			if offset != -1 {

				count++

				var (
					xa = x + xPos*16
					ya = y - 64
					za = z + zPos*16
				)

				buf = buf[:2]
				buf = appendCoord(buf, xa)
				buf = append(buf, ' ')
				buf = appendCoord(buf, ya)
				buf = append(buf, ' ')
				buf = appendCoord(buf, za)
				buf = append(buf, '\n')

				w.Write(buf)
			}
		}
	}
	return
}

func appendFloat(buf []byte, x float64) []byte {
	var highbar float64 = 1.0
	const precision = 32 //match below
	var b [64]byte
	var j = 0

	if x < 0 {
		x = -x
		b[j] = '-'
		j += 1
	}
	for highbar = 1.0; highbar < x; highbar *= 10 {

	}
	numbers := "0123456789"
	for k := j; k < precision; k++ {
		if highbar < 5 && highbar > .5 {
			b[k] = '.'
			k++
		}

		highbar /= 10
		digit := math.Floor(x / highbar)
		b[k] = numbers[int(digit)]
		x -= digit * highbar
	}
	return b[0:precision]
}


func appendCoord(buf []byte, x int) []byte {
	var b [64]byte
	var j = len(b)

	var neg = x < 0
	if neg {
		x = -x
	}

	var (
		high    = x / 20
		low     = (x % 20) * 5
		numbers = "0123456789"
	)

	for i := 0; i < 2; i++ {
		j--
		b[j] = numbers[low%10]
		low /= 10
	}

	j--
	b[j] = '.'

	if high == 0 {
		j--
		b[j] = '0'
	} else {
		for high > 0 {
			j--
			b[j] = numbers[high%10]
			high /= 10
		}
	}

	if neg {
		j--
		b[j] = '-'
	}

	var end = len(buf) + len(b) - j
	var d = buf[len(buf):end]
	copy(d, b[j:])
	return buf[:end]
}

type Vertex struct {
	x, y, z int
}

type blockRun struct {
	blockId        uint16
	v1, v2, v3, v4 Vertex
	dirty          bool
}

func (r *blockRun) AddFace(faces AddFacer) {
	if r.dirty {
		faces.AddFace(r.blockId, r.v1, r.v2, r.v3, r.v4)
		r.dirty = false
	}
}

func (r *blockRun) Update(faces AddFacer, nr *blockRun, flag bool) {
	if !blockFaces {
		if r.dirty {
			if nr.blockId == r.blockId {
				if flag {
					r.v3 = nr.v3
					r.v4 = nr.v4
				} else {
					r.v2 = nr.v2
					r.v3 = nr.v3
				}
			} else {
				r.AddFace(faces)
				*r = *nr
			}
		} else {
			*r = *nr
		}
	} else {
		nr.AddFace(faces)
		nr.dirty = false
	}
}

func (fs *Faces) processBlocks(enclosedChunk *EnclosedChunk, faces AddFacer) {
	for i := 0; i < len(enclosedChunk.blocks); i += 128 {
		var x, z = (i / 128) / 16, (i / 128) % 16

		var r1, r2, r3, r4 blockRun

		var column = BlockColumn(enclosedChunk.blocks[i : i+128])
		for y, blockId := range column {
			if y < yMin {
				continue
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x, y-1, z)) {
				faces.AddFace(blockId, Vertex{x, y, z}, Vertex{x + 1, y, z}, Vertex{x + 1, y, z + 1}, Vertex{x, y, z + 1})
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x, y+1, z)) {
				faces.AddFace(blockId, Vertex{x, y + 1, z}, Vertex{x, y + 1, z + 1}, Vertex{x + 1, y + 1, z + 1}, Vertex{x + 1, y + 1, z})
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x-1, y, z)) {
				r1.Update(faces, &blockRun{blockId, Vertex{x, y, z}, Vertex{x, y, z + 1}, Vertex{x, y + 1, z + 1}, Vertex{x, y + 1, z}, true}, true)
			} else {
				r1.AddFace(faces)
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x+1, y, z)) {
				r2.Update(faces, &blockRun{blockId, Vertex{x + 1, y, z}, Vertex{x + 1, y + 1, z}, Vertex{x + 1, y + 1, z + 1}, Vertex{x + 1, y, z + 1}, true}, false)
			} else {
				r2.AddFace(faces)
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x, y, z-1)) {
				r3.Update(faces, &blockRun{blockId, Vertex{x, y, z}, Vertex{x, y + 1, z}, Vertex{x + 1, y + 1, z}, Vertex{x + 1, y, z}, true}, false)
			} else {
				r3.AddFace(faces)
			}

			if fs.boundary.IsBoundary(blockId, enclosedChunk.Get(x, y, z+1)) {
				r4.Update(faces, &blockRun{blockId, Vertex{x, y, z + 1}, Vertex{x + 1, y, z + 1}, Vertex{x + 1, y + 1, z + 1}, Vertex{x, y + 1, z + 1}, true}, true)
			} else {
				r4.AddFace(faces)
			}
		}

		r1.AddFace(faces)
		r2.AddFace(faces)
		r3.AddFace(faces)
		r4.AddFace(faces)
	}
}
