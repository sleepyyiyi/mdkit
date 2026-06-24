// Command mdkit is the unified CLI entry point for the Markdown toolkit.
//
// Subcommands:
//
//	mdkit convert  input.md              Markdown→sanitized HTML (stdout)
//	mdkit convert  input.md -o out.html  Markdown→sanitized HTML (file)
//	mdkit convert  -                     read Markdown from stdin
//	mdkit summarize input.md             document summary (stdout)
//	mdkit summarize input.md -n 20       summary capped at 20 words
//	mdkit serve                          start HTTP server (:8080)
//	mdkit serve -addr :3000              start on custom port
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"mdkit/internal/health"
	"mdkit/internal/markdown"
)

const version = "0.1.0"

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "convert":
		runConvert(os.Args[2:])
	case "summarize":
		runSummarize(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println("mdkit", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		// Convenience: if the first arg looks like a .md file, treat it as "convert".
		if looksLikeFile(os.Args[1]) {
			runConvert(os.Args[1:])
		} else {
			log.Fatalf("unknown command: %s\nRun 'mdkit help' for usage.", os.Args[1])
		}
	}
}

func runConvert(args []string) {
	fs := flag.NewFlagSet("convert", flag.ExitOnError)
	outFile := fs.String("o", "", "write HTML to file instead of stdout")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: mdkit convert <input.md> [-o output.html]")
		fmt.Fprintln(os.Stderr, "       mdkit convert -            (read from stdin)")
		fs.PrintDefaults()
	}
	// Go's flag stops at the first non-flag arg, so reorder to let flags
	// appear before or after the positional file argument.
	fs.Parse(reorderFlags(args))

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}

	src, err := readInput(fs.Arg(0))
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	svc := markdown.NewService(markdown.NewMockLLM())
	resp, err := svc.Convert(src)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if *outFile != "" {
		if err := os.WriteFile(*outFile, []byte(wrapHTML(resp.HTML)), 0644); err != nil {
			log.Fatalf("error writing %s: %v", *outFile, err)
		}
		fmt.Fprintf(os.Stderr, "✓ wrote %s (%d bytes HTML)\n", *outFile, resp.Bytes)
	} else {
		fmt.Print(resp.HTML)
	}
}

func runSummarize(args []string) {
	fs := flag.NewFlagSet("summarize", flag.ExitOnError)
	maxWords := fs.Int("n", 40, "maximum words in summary")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: mdkit summarize <input.md> [-n maxWords]")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}

	src, err := readInput(fs.Arg(0))
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	svc := markdown.NewService(markdown.NewMockLLM())
	resp, err := svc.Summarize(context.Background(), src, *maxWords)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Println(resp.Summary)
	if !resp.AIAvailable {
		fmt.Fprintln(os.Stderr, "⚠ AI unavailable, used extractive fallback")
	}
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "listen address")
	fs.Parse(args)

	mux := http.NewServeMux()
	healthSvc := health.NewService(version)
	mux.HandleFunc("GET /healthz", health.Handler(healthSvc))

	mdSvc := markdown.NewService(markdown.NewMockLLM())
	markdown.RegisterRoutes(mux, mdSvc)

	log.Printf("mdkit serving on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

// readInput reads from a file path or stdin (when path is "-").
func readInput(path string) (string, error) {
	var r io.Reader
	if path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("cannot open %s: %w", path, err)
		}
		defer f.Close()
		r = f
	}

	data, err := io.ReadAll(io.LimitReader(r, int64(markdown.MaxInputBytes)+1))
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	if len(data) > markdown.MaxInputBytes {
		return "", fmt.Errorf("input exceeds %d bytes limit", markdown.MaxInputBytes)
	}
	return string(data), nil
}

// wrapHTML wraps a body fragment in a minimal standalone HTML document so the
// output can be opened directly in a browser.
func wrapHTML(body string) string {
	return `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>mdkit output</title>
<style>
  body { max-width: 48em; margin: 2em auto; padding: 0 1em; font-family: system-ui, sans-serif; line-height: 1.6; color: #333; }
  pre { background: #f5f5f5; padding: 1em; overflow-x: auto; border-radius: 4px; }
  code { background: #f0f0f0; padding: 0.2em 0.4em; border-radius: 3px; font-size: 0.9em; }
  pre code { background: none; padding: 0; }
  blockquote { border-left: 4px solid #ddd; margin: 1em 0; padding: 0.5em 1em; color: #666; }
  a { color: #0366d6; }
</style>
</head>
<body>
` + body + `</body>
</html>
`
}

// reorderFlags moves flag-like args (e.g. -o val) before positional args so
// Go's flag package parses them correctly regardless of user ordering.
// "mdkit convert README.md -o out.html" → ["-o", "out.html", "README.md"]
func reorderFlags(args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		if len(args[i]) > 0 && args[i][0] == '-' && args[i] != "-" {
			flags = append(flags, args[i])
			// consume the next arg as the flag value if this is a -key form (not --key=val)
			if !containsEqual(args[i]) && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return append(flags, positional...)
}

func containsEqual(s string) bool {
	for _, c := range s {
		if c == '=' {
			return true
		}
	}
	return false
}

func looksLikeFile(s string) bool {
	n := len(s)
	return n > 3 && (s[n-3:] == ".md" || (n > 9 && s[n-9:] == ".markdown"))
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `mdkit — 安全 Markdown→HTML 转换工具

Usage:
  mdkit convert  <input.md> [-o output.html]   转换为消毒后的 HTML
  mdkit convert  -                             从 stdin 读取
  mdkit summarize <input.md> [-n maxWords]     生成文档摘要
  mdkit serve [-addr :8080]                    启动 HTTP 服务
  mdkit version                                显示版本
  mdkit help                                   显示此帮助

Examples:
  mdkit convert README.md                      输出 HTML 到终端
  mdkit convert README.md -o readme.html       生成可浏览器打开的 HTML 文件
  cat doc.md | mdkit convert -                 管道输入
  mdkit summarize README.md -n 20              20 词以内的摘要
  mdkit serve -addr :3000                      在 3000 端口启动 API`)
}
