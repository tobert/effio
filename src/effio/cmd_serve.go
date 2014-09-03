package effio

/*
 * Copyright 2014 Albert P. Tobey <atobey@datastax.com> @AlTobey
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (cmd *Cmd) ServeHTTP() {
	var addrFlag string

	cmd.DefaultFlags()
	cmd.FlagSet.StringVar(&addrFlag, "addr", ":9000", "IP:PORT or :PORT address to listen on")
	cmd.ParseArgs()

	if cmd.PathFlag == "" {
		cmd.PathFlag = "./suites"
	}

	http.HandleFunc("/inventory", cmd.InventoryHandler)
	http.Handle("/", http.FileServer(http.Dir("./public")))

	err := http.ListenAndServe(addrFlag, nil)
	if err != nil {
		log.Fatalf("net.http could not listen on address '%s': %s\n", addrFlag, err)
	}
}

func (cmd *Cmd) InventoryHandler(w http.ResponseWriter, r *http.Request) {
	items := InventorySuiteData(cmd.PathFlag, ".json")

	json, err := json.Marshal(items)
	if err != nil {
		log.Printf("JSON marshal failed: %s\n", err)
		http.Error(w, fmt.Sprintf("Marshaling JSON failed: %s", err), 500)
	}

	w.Write(json)
}

func InventorySuiteData(dpath string, wantSuffix string) []string {
	out := make([]string, 0)

	visitor := func(dpath string, f os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf("Encountered an error while inventorying data in '%s': %s", dpath, err)
		}

		if strings.HasSuffix(dpath, wantSuffix) {
			out = append(out, dpath)
		}

		return nil
	}

	err := filepath.Walk(dpath, visitor)
	if err != nil {
		log.Fatalf("Could not inventory suites in '%s': %s", dpath, err)
	}

	return out
}
