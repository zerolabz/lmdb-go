package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/glycerine/lmdb-go/lmdb"
)

// This example shows the deadlock under linux without the use of the
// fix provided by  -DMDB_USE_SYSV_SEM
//
// from https://gist.githubusercontent.com/xlab/f7aee266ab741a0412b44ee145ccbc23/raw/035e6af42fedcc9f1a2d12013ca480a982df914a/lmdb_deadlock.go
//
// aka https://gist.github.com/xlab/f7aee266ab741a0412b44ee145ccbc23
//
/*
// discussion in https://github.com/bmatsuo/lmdb-go/issues/94

xlab commented on Jan 10, 2017 •
Not sure a branch that deadlocks on popular linux distros can be considered "stable", especially where there is no found traits from the "unstable" one.

When I first discovered that issue months ago, I spent 2 days debugging, just by a wild guess and by reading mdb.c code I found the root cause. I think you can search for MDB_USE_SYSV_SEM in mdb.c too, there are some comments about safety, robustness and availability of this backend in lmdb.h.


xlab commented on Jan 10, 2017 •
Anyway, talk is cheap. There is a minimal reproducible example:
https://gist.github.com/xlab/f7aee266ab741a0412b44ee145ccbc23

$ go get gist.github.com/xlab/f7aee266ab741a0412b44ee145ccbc23.git
$ f7aee266ab741a0412b44ee145ccbc23.git
sync batch 0
sync batch 64000
sync batch 128000
...
sync batch 960000
last put took: 46.385007ms
last key: 001000000
done
The deadlock happens there https://gist.github.com/xlab/f7aee266ab741a0412b44ee145ccbc23#file-lmdb_deadlock-go-L40 with random chance of occurrence, so life is shiny and fun.

The example runs successfully under OS X or on AWS Linux with -DMDB_USE_SYSV_SEM=1

xlab commented on Jan 10, 2017 •
The reason why it happens and why the switch makes sense is because of the location where semaphores are kept. POSIX are kept in thread local storage, and SysV are system-wide. See http://stackoverflow.com/questions/368322/differences-between-system-v-and-posix-semaphores


bmatsuo commented on Jan 29, 2017 •

But yes, write transactions must be confined to a single goroutine which has called LockOSThread(). You can use different goroutines for different write transactions, you just have to ensure that each goroutine that wants to write to the database lock its thread before doing so and never lets the Txn object's methods be called from another goroutine.

*/

func main() {
	initDir("test.db")
	env := openEnv("test.db", lmdb.NoTLS)
	defer env.Close()

	checkErr(env.SetMapSize(1024 * 1024 * 1024))
	txn, err := env.BeginTxn(nil, 0)
	checkErr(err)
	dbi, err := txn.OpenDBI("default", lmdb.Create)
	checkErr(err)

	var t0 time.Time
	var batch = 64000
	var limit = 1000000
	var val = []byte("0000000000000000000000000")

	for i := 0; i < limit; i++ {
		if i == limit-1 {
			// record time of the last run
			t0 = time.Now()
		}

		key := makeKey()
		checkErr(txn.Put(dbi, key, val, 0))

		if i%batch == 0 {
			log.Println("sync batch", i)
			checkErr(txn.Commit())
			txn, err = env.BeginTxn(nil, 0)
			checkErr(err)
			dbi, err = txn.OpenDBI("default", 0)
			checkErr(err)
		}

		if i == limit-1 {
			checkErr(txn.Commit())
			// print the duration of the last run
			log.Println("last put took:", time.Now().Sub(t0))
			log.Println("last key:", string(key))
		}
	}

	log.Println("done")
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln("[ERR]", err)
	}
}

var clock int

func makeKey() []byte {
	clock++
	return []byte(fmt.Sprintf("%09d", clock))
}

func initDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func openEnv(db string, flags uint) *lmdb.Env {
	env, err := lmdb.NewEnv()
	if err != nil {
		log.Fatalln(err)
	}
	if err := env.SetMaxDBs(1024); err != nil {
		log.Fatalln(err)
	}
	if _, err := os.Stat(db); err != nil && os.IsNotExist(err) {
		if err := initDir(db); err != nil {
			log.Fatalln(err)
		}
	}
	if err := env.Open(db, flags, 0644); err != nil {
		log.Fatalln(err)
	}
	return env
}
