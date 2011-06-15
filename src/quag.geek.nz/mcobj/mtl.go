package main

import (
	"fmt"
	"io"
	"os"
	"unicode"
	"strings"
)

func getMtlName(blockId uint16, repeating bool) (retval string) {
	if useTextures==false && noColor==true {
		retval = ""
	} else if noColor {
		placeholder := RepetitionBlockIdNamer{};
		retval = placeholder.NameBlockId(blockId, repeating);
	} else {
		retval = MaterialNamer.NameBlockId(blockId,repeating)
	}
	return
}

func printMtl(w io.Writer, blockId uint16, repeating bool) (retval string) {
	if useTextures==false&&noColor==true {
		retval = ""
	} else {
		retval = getMtlName(blockId, repeating)
		fmt.Fprintln(w, "usemtl "+retval)
	}
	return
}
func returnAlphaNum(rune int) int {
	if unicode.IsLetter(rune) || unicode.IsDigit(rune) {
		return rune;
	}
	return -1;
}
func RepeatingMaterialName(textureName string) string {
	return strings.Map(returnAlphaNum,textureName);
}
func writeMtlFile(filename string) os.Error {
	if noColor && useTextures==false{
		return nil
	}

	var outFile, outErr = os.Create(filename)
	if outErr != nil {
		return outErr
	}
	defer outFile.Close()
	repeatingTextures := make (map [string]bool);
	for _, blockType := range blockTypeMap {
		for _, color := range blockType.colors {
			repeatingTextures [color.Print(outFile)]=true
		}
	}
	fmt.Fprintf(outFile,"newmtl %s\nmap_Kd terrain.png\n\n", "Default")
	for repeatingTexture,_ := range repeatingTextures {
		fmt.Fprintf(outFile,"newmtl %s\nmap_Kd %s\n\n", RepeatingMaterialName(repeatingTexture), repeatingTexture)
	}
	return nil
}

type Vec2 struct {
	x float32
	y float32
}

type TexCoord struct {
	topLeft     Vec2
	bottomRight Vec2
}

func NullTexCoord() TexCoord {
	return TexCoord{Vec2{0, 0},
		Vec2{0, 0}}
}
func (t TexCoord) equals(u TexCoord) bool {
	return t.topLeft.x == u.topLeft.x &&
		t.topLeft.y == u.topLeft.y &&
		t.bottomRight.x == u.bottomRight.x &&
		t.bottomRight.y == u.bottomRight.y

}

func (t TexCoord) isNull() bool {
	return t.equals(NullTexCoord())
}

func NewTexCoord(v00 float64, v01 float64, v10 float64, v11 float64) TexCoord {
	return TexCoord{Vec2{float32(v00), float32(v01)},
		Vec2{float32(v10), float32(v11)}}

}

func (t TexCoord) TopLeft() Vec2 {
	return t.topLeft
}

func (t TexCoord) BottomRight() Vec2 {
	return t.bottomRight
}

func (t TexCoord) TopRight() Vec2 {
	return Vec2{t.bottomRight.x, t.topLeft.y}
}

func (t TexCoord) BottomLeft() Vec2 {
	return Vec2{t.topLeft.x, t.bottomRight.y}
}

func (t TexCoord) vertex(i int) Vec2 {
	switch i {
	case 0:
		return t.TopLeft()
	case 1:
		return t.TopRight()
	case 2:
		return t.BottomRight()
	case 3:
		return t.BottomLeft()
	}
	return Vec2{0, 0}
}


type MTL struct {
	blockId              byte
	metadata             byte
	color                uint32
	name                 string
	repeatingTextureName string //which texture holds the repeating block
	repeatingSideOffset  TexCoord
	repeatingFrontOffset TexCoord
	sideTex              TexCoord
	frontTex             TexCoord
	topTex               TexCoord
	botTex               TexCoord
}

func (mtl *MTL) Print(w io.Writer) (repeating_texturename string){
	var (
		r = mtl.color >> 24
		g = mtl.color >> 16 & 0xff
		b = mtl.color >> 8 & 0xff
		a = mtl.color & 0xff
	)

	fmt.Fprintf(w, "# %s\nnewmtl %s\nKd %.4f %.4f %.4f\nd %.4f\nillum 1\n", mtl.name, MaterialNamer.NameBlockId(uint16(mtl.blockId)+uint16(mtl.metadata)*256,false), float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255)
	if useTextures {
		fmt.Fprintf(w,"map_Kd terrain.png\n\n");
	}
	fmt.Fprintf(w, "# %s\nnewmtl %s\nKd %.4f %.4f %.4f\nd %.4f\nillum 1\n",mtl.name, MaterialNamer.NameBlockId(uint16(mtl.blockId)+uint16(mtl.metadata)*256,true), float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255);
	if useTextures {
																																																								  fmt.Fprintf(w,"map_Kd %s\n\n", mtl.repeatingTextureName);
}
	repeating_texturename=mtl.repeatingTextureName;
	return;
}

func (mtl *MTL) colorId() uint16 {
	var id = uint16(mtl.blockId)
	if mtl.metadata != 255 {
		id += uint16(mtl.metadata) << 8
	}
	return id
}

var (
	MaterialNamer BlockIdNamer
)

type BlockIdNamer interface {
	NameBlockId(blockId uint16, repeating bool) string
}

type RepetitionBlockIdNamer struct{}
func (n *RepetitionBlockIdNamer) NameBlockId(blockId uint16, repeating bool) (name string) {
	var idByte = byte(blockId & 0xff)
	name = "Default";
	if !repeating {
		return;
	}

	if blockType, ok := blockTypeMap[idByte]; ok {
		for i, mtl := range blockType.colors {
			if i == 0 || mtl.metadata == uint8(blockId>>8) {
				name = RepeatingMaterialName(mtl.repeatingTextureName);
			}
			if mtl.metadata == uint8(blockId>>8) {
				return
			}
		}
	}
	return
}


type NumberBlockIdNamer struct{}

func (n *NumberBlockIdNamer) NameBlockId(blockId uint16, repeating bool) (name string) {
	var idByte = byte(blockId & 0xff)
	repeatingstr :=""
	if repeating {
		repeatingstr="repeating_"
	}
	name = fmt.Sprintf("%s%d", repeatingstr,idByte)
	defaultName := name

	if blockType, ok := blockTypeMap[idByte]; ok {
		for i, mtl := range blockType.colors {
			if i == 0 || mtl.metadata == uint8(blockId>>8) {
				name = fmt.Sprintf("%%sd_%d", repeatingstr, idByte, (blockId >> 8))
			}
			if mtl.metadata == uint8(blockId>>8) {
				return
			}
			if mtl.metadata == 255 {
				name = defaultName
			}
		}
	}
	return
}

type NameBlockIdNamer struct{}

func (n *NameBlockIdNamer) NameBlockId(blockId uint16, repeating bool) (name string) {
	repeatingstr :=""
	if repeating {
		repeatingstr="repeating_"
	}

	var idByte = byte(blockId & 0xff)
	name = fmt.Sprintf("%sUnknown.%d", repeatingstr, idByte)
	if blockType, ok := blockTypeMap[idByte]; ok {
		for i, color := range blockType.colors {
			if i == 0 || color.blockId == idByte && (color.metadata == 255 || color.metadata == uint8(blockId>>8)) {
				name = repeatingstr+color.name
			}
			if color.blockId == idByte && color.metadata == uint8(blockId>>8) {
				return
			}
		}
	}
	return
}
