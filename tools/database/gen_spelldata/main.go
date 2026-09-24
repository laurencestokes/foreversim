// Regenerates sim/<class>/spell_data_auto_gen.go from the client database.
//
// Its own binary, not a mode of gen_db: gen_db imports every class package, so a class file that
// does not compile stops it. This binary imports sim/core, sim/core/spelldata and sim/common - the
// item-proc routing in tools/database/gen_effects.go reads the store - so the class files are the
// generated files it can rewrite without compiling them, and a broken store stops it the same way a
// broken class file stops gen_db. Neither reaches the tree from here: spelldata_write.go
// type-checks the rendered files in a staging directory and writes none of them until they build -
// unless -unchecked says to write them anyway, for a class just flipped to the store whose call
// sites have not moved off the family table yet.
//
//	go run ./tools/database/gen_spelldata
//	go run ./tools/database/gen_spelldata -check
//	go run ./tools/database/gen_spelldata -unchecked
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/wowsims/forever/tools/database"
)

var dbPath = flag.String("dbPath", "./tools/database/wowsims.db", "Location of the wowsims.db file produced by tools/db2tool")
var check = flag.Bool("check", false, "Name the generated files that are not what this generator writes, and write nothing")
var unchecked = flag.Bool("unchecked", false, "Write the rendered files without type-checking them first, for a class just flipped to the store whose call sites have not moved off the family table yet")

func main() {
	flag.Parse()
	database.DatabasePath = *dbPath

	helper, err := database.NewDBHelper()
	if err != nil {
		log.Fatalf("failed to open %s: %v", *dbPath, err)
	}
	defer helper.Close()

	if *check {
		stale, err := database.CheckSpellDataFiles(helper)
		if err != nil {
			log.Fatalf("failed to generate spell data tables: %v", err)
		}
		for _, path := range stale {
			fmt.Fprintln(os.Stderr, path)
		}
		if len(stale) > 0 {
			os.Exit(1)
		}
		return
	}

	if err := database.GenerateSpellDataFiles(helper, *unchecked); err != nil {
		log.Fatalf("failed to generate spell data tables: %v", err)
	}
}
