
all: msglite msgliteclient

include $(GOROOT)/src/Make.$(GOARCH)


TARG=msglite
GOFILES=\
	core.go\
	httpserver.go\
	httprequest.go\
	server.go\
	stream.go\
	client.go\

CLEANFILES+=msglite
CLEANFILES+=msgliteclient

include $(GOROOT)/src/Make.pkg

main.$O: main.go package
	$(QUOTED_GOBIN)/$(GC) -I_obj $<

msglite: main.$O
	$(QUOTED_GOBIN)/$(LD) -L_obj -o $@ $<

clientmain.$O: clientmain.go package
	$(QUOTED_GOBIN)/$(GC) -I_obj $<

msgliteclient: clientmain.$O
	$(QUOTED_GOBIN)/$(LD) -L_obj -o $@ $<
