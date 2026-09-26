// webfence-report is a local helper for the M2 technical alpha. Private keys
// stay in the native credential store; the verifier is a separate offline tool.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Matte2599/WebFence/internal/reporting"
	"github.com/Matte2599/WebFence/internal/storage"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	ctx := context.Background()
	switch os.Args[1] {
	case "keygen":
		fs := flag.NewFlagSet("keygen", flag.ExitOnError)
		trustPath := fs.String("trust", "", "absolute public trust registry path")
		name := fs.String("name", "", "declared operator name")
		_ = fs.Parse(os.Args[2:])
		require(fs, *trustPath)
		keys := nativeKeys(*trustPath)
		record, err := keys.Generate(ctx, *name)
		fatal(err)
		emit(record)
	case "rotate":
		fs := flag.NewFlagSet("rotate", flag.ExitOnError)
		trustPath := fs.String("trust", "", "absolute trust registry path")
		id := fs.String("kid", "", "current key ID")
		_ = fs.Parse(os.Args[2:])
		require(fs, *trustPath)
		keys := nativeKeys(*trustPath)
		record, err := keys.Rotate(ctx, *id)
		fatal(err)
		emit(record)
	case "revoke":
		fs := flag.NewFlagSet("revoke", flag.ExitOnError)
		trustPath := fs.String("trust", "", "absolute trust registry path")
		id := fs.String("kid", "", "key ID")
		_ = fs.Parse(os.Args[2:])
		require(fs, *trustPath)
		fatal(nativeKeys(*trustPath).Revoke(ctx, *id))
		fmt.Println("revoked")
	case "public":
		fs := flag.NewFlagSet("public", flag.ExitOnError)
		trustPath := fs.String("trust", "", "absolute trust registry path")
		id := fs.String("kid", "", "key ID")
		out := fs.String("out", "", "new absolute public descriptor path")
		_ = fs.Parse(os.Args[2:])
		require(fs, *trustPath, *out)
		trust := openTrust(*trustPath)
		fatal(trust.ExportPublic(*id, *out))
		fmt.Println("public_key_exported")
	case "trust-import":
		fs := flag.NewFlagSet("trust-import", flag.ExitOnError)
		trustPath := fs.String("trust", "", "absolute trust registry path")
		public := fs.String("public", "", "absolute public descriptor path")
		fingerprint := fs.String("fingerprint", "", "independently obtained SHA-256 fingerprint")
		_ = fs.Parse(os.Args[2:])
		require(fs, *trustPath, *public)
		trust := openTrust(*trustPath)
		record, err := trust.ImportTrusted(*public, *fingerprint)
		fatal(err)
		emit(record)
	case "export":
		fs := flag.NewFlagSet("export", flag.ExitOnError)
		dbPath := fs.String("db", "", "absolute project database path")
		runID := fs.String("run", "", "saved run ID")
		out := fs.String("out", "", "new absolute .wfr path")
		trustPath := fs.String("trust", "", "absolute trust registry path for signing")
		id := fs.String("kid", "", "active signer key ID")
		unsigned := fs.Bool("unsigned", false, "make an explicitly unsigned bundle")
		_ = fs.Parse(os.Args[2:])
		require(fs, *dbPath, *out)
		if *runID == "" || (!*unsigned && (!filepath.IsAbs(*trustPath) || *id == "")) || (*unsigned && (*id != "" || *trustPath != "")) {
			usage()
		}
		store, err := storage.Open(ctx, *dbPath)
		fatal(err)
		defer store.Close()
		snapshot, err := reporting.Load(ctx, store, *runID)
		fatal(err)
		var manifest reporting.Manifest
		if *unsigned {
			manifest, err = reporting.Export(ctx, snapshot, *out, reporting.ExportOptions{Offline: true})
		} else {
			manifest, err = nativeKeys(*trustPath).ExportSigned(ctx, snapshot, *out, *id, reporting.ExportOptions{Offline: true})
		}
		fatal(err)
		emit(manifest)
	default:
		usage()
	}
}

func require(fs *flag.FlagSet, paths ...string) {
	if fs.NArg() != 0 {
		usage()
	}
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			usage()
		}
	}
}
func openTrust(path string) *reporting.TrustStore {
	trust, err := reporting.OpenTrustStore(path)
	fatal(err)
	return trust
}
func nativeKeys(path string) *reporting.Keyring {
	keys, err := reporting.NativeKeyring(openTrust(path))
	fatal(err)
	return keys
}
func emit(value any) { encoded, err := json.Marshal(value); fatal(err); fmt.Println(string(encoded)) }
func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func usage() {
	fmt.Fprintln(os.Stderr, "usage: webfence-report {keygen|rotate|revoke|public|trust-import|export} [flags]")
	os.Exit(2)
}
