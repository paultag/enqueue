package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"golang.org/x/exp/inotify"
	"pault.ag/go/config"
	"pault.ag/go/debian/control"
	// "pault.ag/go/mailer"
	"pault.ag/go/reprepro"
)

type Enqueued struct {
	Root      string `flag:"root" description:"Repo root to watch"`
	Templates string `flag:"templates" description:"Mail templates"`
}

func Watch(watcher *inotify.Watcher, file os.FileInfo) error {
	if !file.IsDir() {
		return nil
	}

	incoming := path.Join(file.Name(), "incoming")

	if _, err := os.Stat(incoming); os.IsNotExist(err) {
		return err
	}
	/* Sweep existing files in there */

	if err := watcher.Watch(incoming); err != nil {
		return err
	}

	return nil
}

func Process(changesPath string) {
	repoRoot := path.Clean(path.Join(path.Dir(changesPath), ".."))
	pwd, err := os.Getwd()
	if err != nil {
		log.Printf("%s\n", err)
		return
	}
	gnuPGHome := path.Join(pwd, "..", "private", repoRoot, ".gnupg")
	repo := reprepro.NewRepo(repoRoot, fmt.Sprintf("--gnupghome=%s", gnuPGHome))

	changes, err := control.ParseChangesFile(changesPath)
	if err != nil {
		log.Printf("Error: %s\n", err)
	}

	err = repo.Include(changes.Distribution, changesPath)
	if err != nil {
		log.Printf("Error: %s\n", err)
		changes.Remove()
		log.Printf("Removed %s and associated files\n", changesPath)
	}

	// if err := repo.Export(); err != nil {
	// 	log.Printf("%s\n", err)
	// }
	log.Printf("Included %s into %s", changes.Source, repo.Basedir)
	// log.Printf("%s %s", changesPath, repoRoot)
	changes.Remove()
}

func main() {
	conf := Enqueued{
		Root: ".",
	}

	flags, err := config.LoadFlags("enqueued", &conf)
	if err != nil {
		log.Fatal(err)
	}
	flags.Parse(os.Args[1:])

	files, err := ioutil.ReadDir(conf.Root)
	if err != nil {
		log.Fatal(err)
	}

	watcher, err := inotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if err := Watch(watcher, file); err != nil {
			log.Fatal(err)
		}
	}

	for {
		select {
		case ev := <-watcher.Event:
			if ev.Mask^inotify.IN_CLOSE_WRITE != 0 ||
				!strings.HasSuffix(ev.Name, ".changes") {
				continue
			}
			go Process(ev.Name)
		}
	}
}
