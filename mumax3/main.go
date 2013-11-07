package main

import (
	"flag"
	"github.com/mumax/3/cuda"
	"github.com/mumax/3/engine"
	_ "github.com/mumax/3/ext"
	"github.com/mumax/3/prof"
	"github.com/mumax/3/util"
	"io/ioutil"
	"log"
	"runtime"
	"time"
)

var (
	flag_silent   = flag.Bool("s", false, "Don't generate any log info")
	flag_vet      = flag.Bool("vet", false, "Check input files for errors, but don't run them")
	flag_od       = flag.String("o", "", "Override output directory")
	flag_force    = flag.Bool("f", false, "Force start, clean existing output directory")
	flag_port     = flag.String("http", ":35367", "Port to serve web gui")
	flag_cpuprof  = flag.Bool("cpuprof", false, "Record gopprof CPU profile")
	flag_memprof  = flag.Bool("memprof", false, "Recored gopprof memory profile")
	flag_blocklen = flag.Int("bl", 512, "CUDA 1D thread block length")
	flag_blockX   = flag.Int("bx", 32, "CUDA 2D thread block size X")
	flag_blockY   = flag.Int("by", 32, "CUDA 2D thread block size Y")
	flag_gpu      = flag.Int("gpu", 0, "specify GPU")
	flag_sched    = flag.String("sched", "yield", "CUDA scheduling: auto|spin|yield|sync")
	flag_sync     = flag.Bool("sync", false, "synchronize all CUDA calls (debug)")
	//flag_pagelock = flag.Bool("pagelock", true, "enable CUDA memeory page-locking")
)

func main() {

	defer func() { log.Println("walltime:", time.Since(engine.StartTime)) }()

	flag.Parse()

	engine.DeclFunc("interactive", Interactive, "Wait for GUI interaction")

	log.SetPrefix("")
	log.SetFlags(0)

	if *flag_vet {
		vet()
		return
	}

	if *flag_silent {
		log.SetOutput(ioutil.Discard)
	}

	log.Print("    ", engine.UNAME, "\n")
	log.Print("(c) Arne Vansteenkiste, Dynamat LAB, Ghent University, Belgium", "\n")
	log.Print("    This is free software without any warranty. See license.txt", "\n")
	log.Print("\n")

	if flag.NArg() != 1 {
		batchMode()
		return
	}

	if *flag_od == "" { // -o not set
		engine.SetOD(util.NoExt(flag.Arg(0))+".out", *flag_force)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	cuda.BlockSize = *flag_blocklen
	cuda.TileX = *flag_blockX
	cuda.TileY = *flag_blockY
	cuda.Init(*flag_gpu, *flag_sched, *flag_sync)
	cuda.LockThread()

	if *flag_cpuprof {
		prof.InitCPU(engine.OD)
	}
	if *flag_memprof {
		prof.InitMem(engine.OD)
	}
	defer prof.Cleanup()

	RunFileAndServe(flag.Arg(0))

	keepBrowserAlive() // if open, that is
	engine.Close()
}

// Runs a script file.
func RunFileAndServe(fname string) {
	// first we compile the entire file into an executable tree
	bytes, err := ioutil.ReadFile(fname)
	util.FatalErr(err)
	code, err2 := engine.World.Compile(string(bytes))
	util.FatalErr(err2)

	// now the parser is not used anymore so it can handle web requests
	go engine.Serve(*flag_port)

	// start executing the tree, possibly injecting commands from web gui
	code.Eval()
}
