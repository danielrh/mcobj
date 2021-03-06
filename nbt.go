package nbt

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

const (
	tagStructEnd = 0  // No name. Single zero byte.
	tagInt8      = 1  // A single signed byte (8 bits)
	tagInt16     = 2  // A signed short (16 bits, big endian)
	tagInt32     = 3  // A signed int (32 bits, big endian)
	tagInt64     = 4  // A signed long (64 bits, big endian)
	tagFloat32   = 5  // A floating point value (32 bits, big endian, IEEE 754-2008, binary32)
	tagFloat64   = 6  // A floating point value (64 bits, big endian, IEEE 754-2008, binary64)
	tagByteArray = 7  // { TAG_Int length; An array of bytes of unspecified format. The length of this array is <length> bytes }
	tagString    = 8  // { TAG_Short length; An array of bytes defining a string in UTF-8 format. The length of this array is <length> bytes }
	tagList      = 9  // { TAG_Byte tagId; TAG_Int length; A sequential list of Tags (not Named Tags), of type <typeId>. The length of this array is <length> Tags. } Notes: All tags share the same type.
	tagStruct    = 10 // { A sequential list of Named Tags. This array keeps going until a TAG_End is found.; TAG_End end } Notes: If there's a nested TAG_Compound within this tag, that one will also have a TAG_End, so simply reading until the next TAG_End will not work. The names of the named tags have to be unique within each TAG_Compound The order of the tags is not guaranteed.
)

var (
	ErrListUnknown = os.NewError("Lists of unknown type aren't supported")
)

type Chunk struct {
	XPos, ZPos int
	blocks     []byte
	data       []byte
	Blocks     []uint16
}

func ReadDat(reader io.Reader) (*Chunk, os.Error) {
	var r, rErr = gzip.NewReader(reader)
	defer r.Close()
	if rErr != nil {
		return nil, rErr
	}

	return ReadNbt(r)
}

func ReadNbt(reader io.Reader) (*Chunk, os.Error) {
	return processChunk(bufio.NewReader(reader), false)
}

func processChunk(br *bufio.Reader, readingStruct bool) (*Chunk, os.Error) {
	var chunk *Chunk = nil

	for {
		var typeId, name, err = readTag(br)
		if err != nil {
			if err == os.EOF {
				break
			}
			return chunk, err
		}

		switch typeId {
		case tagStruct:
		case tagStructEnd:
			if readingStruct {
				return chunk, nil
			}
		case tagByteArray:
			var bytes, err2 = readBytes(br)
			if err2 != nil {
				return chunk, err2
			}
			if name == "Blocks" {
				if chunk == nil {
					chunk = new(Chunk)
				}
				chunk.blocks = bytes
			} else if name == "Data" {
				if chunk == nil {
					chunk = new(Chunk)
				}
				chunk.data = bytes
			}
		case tagInt8:
			var _, err2 = readInt8(br)
			if err2 != nil {
				return chunk, err2
			}
		case tagInt16:
			var _, err2 = readInt16(br)
			if err2 != nil {
				return chunk, err2
			}
		case tagInt32:
			var number, err2 = readInt32(br)
			if err2 != nil {
				return chunk, err2
			}

			if chunk == nil {
				chunk = new(Chunk)
			}
			if name == "xPos" {
				chunk.XPos = number
			}
			if name == "zPos" {
				chunk.ZPos = number
			}
		case tagInt64:
			var _, err2 = readInt64(br)
			if err2 != nil {
				return chunk, err2
			}
		case tagFloat32:
			var _, err2 = readInt32(br) // TODO: read floats not ints
			if err2 != nil {
				return chunk, err2
			}
		case tagFloat64:
			var _, err2 = readInt64(br) // TODO: read floats not ints
			if err2 != nil {
				return chunk, err2
			}
		case tagString:
			var _, err2 = readString(br)
			if err2 != nil {
				return chunk, err2
			}
		case tagList:
			var itemTypeId, length, err2 = readListHeader(br)
			if err2 != nil {
				return chunk, err2
			}
			switch itemTypeId {
			case tagInt8:
				for i := 0; i < length; i++ {
					var _, err3 = readInt8(br)
					if err3 != nil {
						return chunk, err3
					}
				}
			case tagFloat32:
				for i := 0; i < length; i++ {
					var _, err3 = readInt32(br) // TODO: read float32 instead of int32
					if err3 != nil {
						return chunk, err3
					}
				}
			case tagFloat64:
				for i := 0; i < length; i++ {
					var _, err3 = readInt64(br) // TODO: read float64 instead of int64
					if err3 != nil {
						return chunk, err3
					}
				}
			case tagStruct:
				var chunk2, err3 = processChunk(br, true)
				if chunk == nil {
					chunk = chunk2
				}
				if err3 != nil {
					return chunk, err3
				}
			default:
				fmt.Printf("# %s list todo(%v) %v\n", name, itemTypeId, length)
				return chunk, ErrListUnknown
			}
		default:
			fmt.Printf("# %s todo(%d)\n", name, typeId)
		}
	}

	if chunk != nil {
		chunk.Blocks = make([]uint16, len(chunk.blocks))
		for i, blockId := range chunk.blocks {
			var metadata byte
			if i&1 == 1 {
				metadata = chunk.data[i/2] >> 4
			} else {
				metadata = chunk.data[i/2] & 0xf
			}
			chunk.Blocks[i] = uint16(blockId) + (uint16(metadata) << 8)
		}
	}

	return chunk, nil
}

func readTag(r *bufio.Reader) (byte, string, os.Error) {
	var typeId, err = r.ReadByte()
	if err != nil || typeId == 0 {
		return typeId, "", err
	}

	var name, err2 = readString(r)
	if err2 != nil {
		return typeId, name, err2
	}

	return typeId, name, nil
}

func readListHeader(r *bufio.Reader) (itemTypeId byte, length int, err os.Error) {
	length = 0

	itemTypeId, err = r.ReadByte()
	if err == nil {
		length, err = readInt32(r)
	}

	return
}

func readString(r *bufio.Reader) (string, os.Error) {
	var length, err1 = readInt16(r)
	if err1 != nil {
		return "", err1
	}

	var bytes = make([]byte, length)
	var _, err2 = io.ReadFull(r, bytes)
	return string(bytes), err2
}

func readBytes(r *bufio.Reader) ([]byte, os.Error) {
	var length, err1 = readInt32(r)
	if err1 != nil {
		return nil, err1
	}

	var bytes = make([]byte, length)
	var _, err2 = io.ReadFull(r, bytes)
	return bytes, err2
}

func readInt8(r *bufio.Reader) (int, os.Error) {
	return readIntN(r, 1)
}

func readInt16(r *bufio.Reader) (int, os.Error) {
	return readIntN(r, 2)
}

func readInt32(r *bufio.Reader) (int, os.Error) {
	return readIntN(r, 4)
}

func readInt64(r *bufio.Reader) (int, os.Error) {
	return readIntN(r, 8)
}

func readIntN(r *bufio.Reader, n int) (int, os.Error) {
	var a int = 0

	for i := 0; i < n; i++ {
		var b, err = r.ReadByte()
		if err != nil {
			return a, err
		}
		a = a<<8 + int(b)
	}

	return a, nil
}
