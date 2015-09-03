package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"

	"golang.org/x/exp/inotify"
	// "pault.ag/go/reprepro"
)

func Watch(watcher *inotify.Watcher, file os.FileInfo) error {
	if !file.IsDir() {
		return nil
	}

	incoming := path.Join(file.Name(), "incoming")

	if _, err := os.Stat(incoming); os.IsNotExist(err) {
		return err
	}

	if err := watcher.Watch(incoming); err != nil {
		return err
	}

	return nil
}

func Process(changesPath string) {
	repoRoot := path.Clean(path.Join(path.Dir(changesPath), ".."))
	// repo := reprepro.NewRepo(repoRoot)
	// repo.Include()
	// log.Printf("%s", repo)
	log.Printf("%s", repoRoot)
}

func main() {
	files, err := ioutil.ReadDir(".")
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
			if ev.Mask^inotify.IN_CLOSE_WRITE != 0 {
				continue
			}
			go Process(ev.Name)
		}
	}
}
