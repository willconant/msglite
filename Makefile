
all: msglite

include $(GOROOT)/src/Make.$(GOARCH)


TARG=msglite
GOFILES=\
	core.go\
	http.go\
	server.go\
	stream.go\

CLEANFILES+=msglite

include $(GOROOT)/src/Make.pkg

main.$O: main.go package
	$(QUOTED_GOBIN)/$(GC) -I_obj $<

msglite: main.$O
	$(QUOTED_GOBIN)/$(LD) -L_obj -o $@ $<
