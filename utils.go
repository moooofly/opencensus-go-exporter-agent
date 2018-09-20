package agent

import (
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"
)

// overflowLogger ensures that at most one overflow error log message is
// written every 5 seconds.
type overflowLogger struct {
	mu    sync.Mutex
	pause bool
	accum int
}

func (o *overflowLogger) delay() {
	o.pause = true
	time.AfterFunc(5*time.Second, func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		switch {
		case o.accum == 0:
			o.pause = false
		case o.accum == 1:
			log.Println("OpenCensus agent exporter: failed to upload span: buffer full")
			o.accum = 0
			o.delay()
		default:
			log.Printf("OpenCensus agent exporter: failed to upload %d spans: buffer full", o.accum)
			o.accum = 0
			o.delay()
		}
	})
}

func (o *overflowLogger) log() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.pause {
		log.Println("OpenCensus agent exporter: failed to upload span: buffer full")
		o.delay()
	} else {
		o.accum++
	}
}

// configRead reads value by specific config key
func ConfigRead(path string, key string) string {
	if path == "" {
		path = DefaultConfigPath
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println("--> find no file: ", path)
		return ""
	}

	content := string(b)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		kv := strings.Split(line, "=")
		if kv[0] == key {
			log.Printf("--> match key[%s], value is: %s\n", key, kv[1])
			return strings.Trim(kv[1], "\"")
		}
	}
	return ""
}
