This is [Gorsium](https://github.com/csabahenk/gorsium), a mean and lean
implementation of the rsync algorithm in [Go](http://golang.org). The most
important thing about it that its name is not just a kind of combination of
"go" and "rsync", but also the name of an
[old Roman settlement](http://en.wikipedia.org/wiki/Tác) located in current
Hungary and you can actually visit the ruins.

Other most important thing is that it's based on the following resources:
[Lars Wirzenius' sketch of the rsync algorithm in Python](http://blog.liw.fi/posts/rsync-in-python/)
and
[description of the rolling (aka. weak) sum in Tridge's thesis](http://rsync.samba.org/tech_report/node3.html).
Lars' writeup is just ingenious beyond words' expressive capability -- there are
a bunch of sketches of rsync in Python but all are lost in technical details
and funky control flow; while Lars' code is a clear demonstration of the
alleged "executable pseudocode" character of Python. Only thing missing bit is
the rolling sum, as that involves a kind of bit-arithmetics with which Python
is pathetic (Lars falls back to another kind of weak sum which is
algorithmically worse than the rolling sum, but available as a C-coded library
module in Python, so it's still faster than if the rolling sum were coded in
pure Python) -- so to pick up that I reached out to Tridge's original
publication.

Now Go... is also quite close to executable pseudocode. Unlike Python, it's
compiled and it's realistic to do bit arithmetics in it. I'm amazed of the
succinctness of the result. Apart for some general goodies of the language
which can be discovered on the homepage, I made a particular good use of the
following features:

- a readily available map (dictionary) type
- general serizalization capabilites
- an excellent rpc module based on the above referred serialization format as
  the wire protocol

Also, if this experient will be continued, I might see a good use of go's
built-in concurrency support.

Compilation
-----------

The code relies on Go (of course), and [Go-gb](http://code.google.com/p/go-gb/) as the
build system. If you start from scratch, without having any of this infrastructure,
the procedure is as follows:

- install [Mercurial](http://mercurial.selenic.com)
- `BASEDIR=$PWD`
- `hg clone -r release.r60.3 https://go.googlecode.com/hg/ go`
- `cd go/src && ./all.bash && cd ../..`
- `git clone -n git://github.com/skelterjohn/go-gb.git`
- `cd go-gb && git checkout go.r60.3`
- `cd gb && PATH=$BASEDIR/go/bin:$PATH gomake && cd ../..`
- we assume gorsium tree is there in `$BASEDIR`
- `cd gorsium && PATH=$BASEDIR/go/bin:$PATH $BASEDIR/go-gb/gb/gb`

Trying out
----------

A basic summary is avaliable by `gorsium -h`. Note that the command line interface
is optimized for machines, designed in a way which is easy both to generate and
parse/validate; so you as a human might find it a bit weird. Therefore, instead
of explaining it in detail, let's see how can you accomplish something with it
that you now already how to do with `rsync`.

- **rsync**: `rsync /foo/a /foo/b example.com:/baz`
- **gorsium**: `gorsium ssh example.com gorsium RECEIVE /baz -- a b SEND /foo`

(contrary to rsync, we should not assume gorsium is in the `$PATH`, so actually
you would pass the full path of the gorsium binaries, so that first one is the
location of the local binary, and second one is the location of the remote
binary (ie. the copy on example.com)).
