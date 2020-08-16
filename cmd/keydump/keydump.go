package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/glycerine/lmdb-go/lmdb"
)

// lumd_keydump simply prints all the keys in the database path specified
// as the first argument on the command line.

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "must supply path to database as only arg\n")
		os.Exit(1)
	}

	path := os.Args[1]
	if !FileExists(path) {
		fmt.Fprintf(os.Stderr, "path '%v' does not exist.\n", path)
		os.Exit(1)
	}

	maxr := 1
	env, err := lmdb.NewEnvMaxReaders(maxr)
	panicOn(err)
	defer env.Close()

	env.SetMapSize(256 << 30)

	err = env.SetMaxDBs(10)
	panicOn(err)

	//var myflags uint = NoReadahead | NoSubdir
	var myflags uint = lmdb.NoSubdir
	err = env.Open(path, myflags, 0664)
	panicOn(err)

	// In any real application it is important to check for readers that were
	// never closed by their owning process, and for which the owning process
	// has exited.  See the documentation on transactions for more information.
	staleReaders, err := env.ReaderCheck()
	panicOn(err)
	if staleReaders > 0 {
		vv("cleared %d reader slots from dead processes", staleReaders)
	}

	dbnames := []string{}

	var dbiRoot lmdb.DBI
	var dbi lmdb.DBI
	env.SphynxReader(func(txn *lmdb.Txn, readslot int) (err error) {
		//txn.RawRead = true

		dbiRoot, err = txn.OpenRoot(0)
		panicOn(err)

		cur, err := txn.OpenCursor(dbiRoot)
		panicOn(err)
		defer cur.Close()

		for i := 0; true; i++ {
			var k, v []byte
			var err error
			if i == 0 {
				// must give it at least a zero byte here to start.
				k, v, err = cur.Get([]byte{0}, nil, lmdb.SetRange)
				panicOn(err)
			} else {
				k, v, err = cur.Get([]byte(nil), nil, lmdb.Next)
				if lmdb.IsNotFound(err) {
					break
				} else {
					panicOn(err)
				}
			}
			dbnames = append(dbnames, string(k))
			_ = v
		}
		cur.Close()
		return
	})

	for _, dbn := range dbnames {
		fmt.Printf(`
=========================
database '%v':
=========================

`, dbn)
		env.SphynxReader(func(txn *lmdb.Txn, readslot int) (err error) {
			//txn.RawRead = true

			dbi, err = txn.OpenDBI(dbn, 0)
			panicOn(err)

			cur, err := txn.OpenCursor(dbi)
			defer cur.Close()

			panicOn(err)

			for i := 0; true; i++ {
				var k, v []byte
				var err error
				if i == 0 {
					// must give it at least a zero byte here to start.
					k, v, err = cur.Get([]byte{0}, nil, lmdb.SetRange)
					panicOn(err)
				} else {
					k, v, err = cur.Get([]byte(nil), nil, lmdb.Next)
					if lmdb.IsNotFound(err) {
						break
					} else {
						panicOn(err)
					}
				}

				vs := ""
				if len(v) < 100 {
					vs = fmt.Sprintf("%x", v) + " "
				}
				fmt.Printf("%04v %v len value; key: '%v' len %v -> %v\n", i, len(v), string(k), len(k), vs)
			}
			return
		})
	} // for dbnames

	fmt.Printf("=================== done.\n")
}
