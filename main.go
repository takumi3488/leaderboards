package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "leaderboards:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out, errOut io.Writer) error {
	if len(args) == 0 {
		return printUsage(out)
	}
	switch args[0] {
	case "help", "-h", "--help":
		return printUsage(out)
	case "fetch":
		flags := flag.NewFlagSet("fetch", flag.ContinueOnError)
		flags.SetOutput(errOut)
		var opts fetchOptions
		flags.StringVar(&opts.DataDir, "data-dir", "data", "directory for original downloaded sources")
		flags.StringVar(&opts.TerminalURL, "terminal-url", defaultTerminalURL, "Terminal-Bench leaderboard page URL")
		flags.StringVar(&opts.FrontierURL, "frontier-url", defaultFrontierURL, "FrontierCode JSON URL")
		flags.StringVar(&opts.SubscriptionURL, "subscription-url", defaultSubscriptionURL, "subscription Markdown URL")
		if err := parseFlags(flags, args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		if err := fetchSources(ctx, opts); err != nil {
			return err
		}
		_, err := fmt.Fprintln(out, "Downloaded original sources to", opts.DataDir)
		return err
	case "build":
		flags := flag.NewFlagSet("build", flag.ContinueOnError)
		flags.SetOutput(errOut)
		dataDir := flags.String("data-dir", "data", "directory containing downloaded sources")
		output := flags.String("output", "leaderboard.html", "single HTML output path")
		if err := parseFlags(flags, args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		r, err := loadReport(*dataDir)
		if err != nil {
			return err
		}
		if err := writeReport(*output, r); err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, "Wrote", *output)
		return err
	default:
		return fmt.Errorf("unknown command %q; use fetch, build, or help", args[0])
	}
}

func parseFlags(flags *flag.FlagSet, args []string) error {
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	return nil
}

func printUsage(out io.Writer) error {
	_, err := fmt.Fprintln(out, `Usage:
  go run . fetch [-data-dir data]
                [-terminal-url URL] [-frontier-url URL] [-subscription-url URL]
  go run . build [-data-dir data] [-output leaderboard.html]

fetch saves the original benchmark responses and latest subscription Markdown.
build reads only local files. Viewing the chart requires the ECharts CDN.
Use fetch -h or build -h for flag details.`)
	return err
}

func loadReport(dataDir string) (report, error) {
	benchmarks, sources, err := loadBenchmarks(dataDir)
	if err != nil {
		return report{}, err
	}
	path := filepath.Join(dataDir, subscriptionMarkdownFile)
	markdown, err := os.ReadFile(path)
	if err != nil {
		return report{}, fmt.Errorf("read subscription source (run fetch first): %w", err)
	}
	subscriptions, updated, err := parseSubscriptions(markdown)
	if err != nil {
		return report{}, fmt.Errorf("parse %s: %w", path, err)
	}
	applySubscriptions(benchmarks, subscriptions)
	return report{
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		SubscriptionUpdated: updated,
		Benchmarks:          benchmarks,
		Subscriptions:       subscriptions,
		Sources:             sources,
	}, nil
}

func writeReport(path string, r report) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".leaderboard-*.html")
	if err != nil {
		return fmt.Errorf("create HTML output: %w", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if err := renderHTML(f, r); err != nil {
		_ = f.Close()
		return fmt.Errorf("render HTML: %w", err)
	}
	// CreateTemp uses 0600; Pages and local viewers need a normally readable file.
	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()
		return fmt.Errorf("set HTML permissions: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close HTML: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("save HTML: %w", err)
	}
	return nil
}
