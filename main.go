package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type dorkSource struct {
	start string
	end   string
}

var sources = map[string]dorkSource{
	"github":  {start: "https://github.com/search?q=%22", end: "&type=Code"},
	"google":  {start: "https://www.google.com/search?q=%22", end: "&num=100"},
	"shodan":  {start: "https://www.shodan.io/search?query=%22", end: ""},
	"wayback": {start: "https://web.archive.org/web/*/", end: ""},
}

const defaultOutputPermission = 0o644

func main() {
	orgName := flag.String("organization", "", "Specify a single organization.")
	listPath := flag.String("list", "", "Path to a file with a list of organizations.")
	outputPrefix := flag.String("output", "", "Word to prepend to generated output filenames.")
	outputGitHub := flag.String("output-github", "", "File to write GitHub dork links.")
	outputGoogle := flag.String("output-google", "", "File to write Google dork links.")
	outputShodan := flag.String("output-shodan", "", "File to write Shodan dork links.")
	outputWayback := flag.String("output-wayback", "", "File to write Wayback Machine dork links.")
	wordlist := flag.String("w", "", "Path to wordlist used for all enabled dork types.")
	var (
		wordlistGitHub  string
		wordlistGoogle  string
		wordlistShodan  string
		wordlistWayback string
	)
	flag.StringVar(&wordlistGitHub, "wordlist-github", "", "Path to GitHub wordlist.")
	flag.StringVar(&wordlistGitHub, "wGh", "", "Short alias for --wordlist-github.")
	flag.StringVar(&wordlistGoogle, "wordlist-google", "", "Path to Google wordlist.")
	flag.StringVar(&wordlistGoogle, "wGg", "", "Short alias for --wordlist-google.")
	flag.StringVar(&wordlistShodan, "wordlist-shodan", "", "Path to Shodan wordlist.")
	flag.StringVar(&wordlistShodan, "wSh", "", "Short alias for --wordlist-shodan.")
	flag.StringVar(&wordlistWayback, "wordlist-wayback", "", "Path to Wayback wordlist.")
	flag.StringVar(&wordlistWayback, "wWb", "", "Short alias for --wordlist-wayback.")
	generateAll := flag.Bool("all", false, "Generate all dork types (default).")
	generateGitHub := flag.Bool("github", false, "Generate GitHub dork links.")
	generateGoogle := flag.Bool("google", false, "Generate Google dork links.")
	generateShodan := flag.Bool("shodan", false, "Generate Shodan dork links.")
	generateWayback := flag.Bool("wayback", false, "Generate Wayback Machine dork links.")

	flag.Parse()

	if *orgName == "" && *listPath == "" {
		fail("provide either --organization or --list")
	}

	types := resolveTypes(*generateAll, *generateGitHub, *generateGoogle, *generateShodan, *generateWayback)

	generalWordlist := expandHome(*wordlist)
	wordlists := map[string]string{
		"github":  resolveWordlist(wordlistGitHub, generalWordlist),
		"google":  resolveWordlist(wordlistGoogle, generalWordlist),
		"shodan":  resolveWordlist(wordlistShodan, generalWordlist),
		"wayback": resolveWordlist(wordlistWayback, generalWordlist),
	}

	for _, dtype := range types {
		if wordlists[dtype] == "" {
			fail("provide a wordlist for %s via -w or --wordlist-%s", dtype, dtype)
		}
	}

	targetDir := os.Getenv("TARGET")
	if targetDir == "" {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			fail("unable to determine working directory: %v", err)
		}
	}

	dorkingDir := os.Getenv("DORKING")
	if dorkingDir == "" {
		dorkingDir = filepath.Join(targetDir, "dorking")
	}
	dorkingDir = expandHome(dorkingDir)

	if err := os.MkdirAll(dorkingDir, 0o755); err != nil {
		fail("create dorking directory: %v", err)
	}

	outputFiles := map[string]string{
		"github":  *outputGitHub,
		"google":  *outputGoogle,
		"shodan":  *outputShodan,
		"wayback": *outputWayback,
	}

	processOrganization := func(org string) {
		org = strings.TrimSpace(org)
		if org == "" {
			return
		}

		for _, dtype := range types {
			wordlist := wordlists[dtype]
			if wordlist == "" {
				continue
			}
			if !fileExists(wordlist) {
				fmt.Fprintf(os.Stderr, "wordlist missing for %s: %s\n", dtype, wordlist)
				continue
			}

			outputPath := outputFiles[dtype]
			if outputPath == "" {
				outputPath = defaultOutputPath(dorkingDir, org, dtype, *outputPrefix)
			} else {
				outputPath = expandHome(outputPath)
			}

			if err := generateLinks(wordlist, outputPath, dtype, org); err != nil {
				fmt.Fprintf(os.Stderr, "generate %s links for %s: %v\n", dtype, org, err)
			}
		}
	}

	if *listPath != "" {
		f, err := os.Open(expandHome(*listPath))
		if err != nil {
			fail("open organization list: %v", err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			processOrganization(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fail("read organization list: %v", err)
		}
	} else {
		processOrganization(*orgName)
	}
}

func resolveTypes(all, github, google, shodan, wayback bool) []string {
	if all || (!github && !google && !shodan && !wayback) {
		return []string{"github", "google", "shodan", "wayback"}
	}
	types := make([]string, 0, 4)
	if github {
		types = append(types, "github")
	}
	if google {
		types = append(types, "google")
	}
	if shodan {
		types = append(types, "shodan")
	}
	if wayback {
		types = append(types, "wayback")
	}
	return types
}

func generateLinks(wordlistFile, outputFile, dtype, org string) error {
	if fileExists(outputFile) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		return err
	}

	in, err := os.Open(wordlistFile)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, defaultOutputPermission)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	defer out.Close()

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		encoded := url.QueryEscape(line)
		dork := buildDorkURL(dtype, encoded, org)
		if _, err := fmt.Fprintln(out, dork); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func buildDorkURL(dtype, encodedTerm, org string) string {
	switch dtype {
	case "github":
		return fmt.Sprintf("%s%s%%22+%%22%s%%22%s", sources[dtype].start, encodedTerm, org, sources[dtype].end)
	case "google":
		return fmt.Sprintf("%s%s%%22+site:%s%s", sources[dtype].start, encodedTerm, org, sources[dtype].end)
	case "shodan":
		return fmt.Sprintf("%s%s+hostname:\"%s\"%s", sources[dtype].start, encodedTerm, org, sources[dtype].end)
	case "wayback":
		return fmt.Sprintf("%s%s%%20%s%s", sources[dtype].start, org, encodedTerm, sources[dtype].end)
	default:
		return ""
	}
}

func defaultOutputPath(baseDir, org, dtype, prefix string) string {
	name := org
	if prefix != "" {
		name = fmt.Sprintf("%s_%s", prefix, org)
	}
	return filepath.Join(baseDir, fmt.Sprintf("%s_%s_dork_links.txt", name, dtype))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

func resolveWordlist(specific, general string) string {
	if strings.TrimSpace(specific) != "" {
		return expandHome(specific)
	}
	return general
}
