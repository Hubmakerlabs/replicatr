# Installing Go

Go 1.2+ is recommended - unlike most other languages, the forward compatibility
guarantee is ironclad, so go to [https://go.dev/dl/](https://go.dev/dl/) and
pick the latest one (1.22.3 at time of writing), "copy link location" on the
relevant version (linux x86-64 in this example, which applies to Linux and WSL, for Mac [see here](https://go.dev/dl/) -
not tested for BSDs or Windows but should work).

```bash
cd
mkdir bin 
wget https://go.dev/dl/go1.22.3.linux-amd64.tar.gz
tar xvf go1.18.linux-amd64.tar.gz
```

Using your favourite editor, open up `~/.bashrc` - or just

```bash
nano ~/.bashrc
```

and put the following lines at the end

```bash
export GOBIN=$HOME/bin
export GOPATH=$HOME
export GOROOT=$GOPATH/go
export PATH=$HOME/go/bin:$HOME/.local/bin:$GOBIN:$PATH
``` 

save and close, and `ctrl-d` to kill the terminal session, and start a new one.

This also creates a proper place where `go install` will put produced binaries.
