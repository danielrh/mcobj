include $(GOROOT)/src/Make.inc

TARG=mcobj
GOFILES=\
	mcobj.go\
	texture.go\
	version.go\
	obj.go\
	objparse.go\
	mtl.go\
	faces.go\
	prt.go\
	sides.go\
	sideCache.go\
	enclosedChunk.go\
	world.go\
	blocktypes.go\
	alphaworld.go\
	betaworld.go\
	chunkmasks.go\
	usage-$(GOOS).go\

include $(GOROOT)/src/Make.cmd
