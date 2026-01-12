package benchmarks

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// BenchmarkResult represents a single benchmark result.
type BenchmarkResult struct {
	Name        string
	Iterations  int
	NsPerOp     float64
	MBPerSec    float64
	BytesPerOp  int64
	AllocsPerOp int64
}

// ParseBenchmarkOutput parses Go benchmark output and returns structured results.
func ParseBenchmarkOutput(r io.Reader) ([]BenchmarkResult, error) {
	scanner := bufio.NewScanner(r)
	var results []BenchmarkResult

	// Regex to parse benchmark lines
	// Example: BenchmarkVectorStore_Memory/Upsert/Batch10-8   100000   10234 ns/op   1234 B/op   12 allocs/op
	benchRegex := regexp.MustCompile(`^Benchmark(\S+)\s+(\d+)\s+(\d+\.?\d*)\s+ns/op(?:\s+(\d+\.?\d*)\s+MB/s)?(?:\s+(\d+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := benchRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		result := BenchmarkResult{
			Name: matches[1],
		}

		_, _ = fmt.Sscanf(matches[2], "%d", &result.Iterations)
		_, _ = fmt.Sscanf(matches[3], "%f", &result.NsPerOp)

		if matches[4] != "" {
			_, _ = fmt.Sscanf(matches[4], "%f", &result.MBPerSec)
		}
		if matches[5] != "" {
			_, _ = fmt.Sscanf(matches[5], "%d", &result.BytesPerOp)
		}
		if matches[6] != "" {
			_, _ = fmt.Sscanf(matches[6], "%d", &result.AllocsPerOp)
		}

		results = append(results, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading benchmark output: %w", err)
	}

	return results, nil
}

// GenerateMarkdownReport creates a markdown report from benchmark results.
func GenerateMarkdownReport(results []BenchmarkResult) string {
	var sb strings.Builder

	sb.WriteString("# Benchmark Results\n\n")
	sb.WriteString(fmt.Sprintf("Total benchmarks: %d\n\n", len(results)))

	// Group by category
	categories := make(map[string][]BenchmarkResult)
	for _, r := range results {
		parts := strings.Split(r.Name, "/")
		category := parts[0]
		categories[category] = append(categories[category], r)
	}

	// Sort categories
	var categoryNames []string
	for name := range categories {
		categoryNames = append(categoryNames, name)
	}
	sort.Strings(categoryNames)

	// Generate tables for each category
	for _, category := range categoryNames {
		results := categories[category]
		sb.WriteString(fmt.Sprintf("## %s\n\n", category))
		sb.WriteString("| Benchmark | Iterations | ns/op | MB/s | B/op | allocs/op |\n")
		sb.WriteString("|-----------|------------|-------|------|------|----------|\n")

		for _, r := range results {
			mbPerSec := "-"
			if r.MBPerSec > 0 {
				mbPerSec = fmt.Sprintf("%.2f", r.MBPerSec)
			}

			bytesPerOp := "-"
			if r.BytesPerOp > 0 {
				bytesPerOp = fmt.Sprintf("%d", r.BytesPerOp)
			}

			allocsPerOp := "-"
			if r.AllocsPerOp > 0 {
				allocsPerOp = fmt.Sprintf("%d", r.AllocsPerOp)
			}

			sb.WriteString(fmt.Sprintf("| %s | %d | %.2f | %s | %s | %s |\n",
				r.Name, r.Iterations, r.NsPerOp, mbPerSec, bytesPerOp, allocsPerOp))
		}
		sb.WriteString("\n")
	}

	// Add summary statistics
	sb.WriteString("## Summary\n\n")

	var totalNs float64
	var fastest, slowest BenchmarkResult
	for i, r := range results {
		totalNs += r.NsPerOp
		if i == 0 || r.NsPerOp < fastest.NsPerOp {
			fastest = r
		}
		if i == 0 || r.NsPerOp > slowest.NsPerOp {
			slowest = r
		}
	}

	avgNs := totalNs / float64(len(results))
	sb.WriteString(fmt.Sprintf("- **Average latency**: %.2f ns/op\n", avgNs))
	sb.WriteString(fmt.Sprintf("- **Fastest**: %s (%.2f ns/op)\n", fastest.Name, fastest.NsPerOp))
	sb.WriteString(fmt.Sprintf("- **Slowest**: %s (%.2f ns/op)\n", slowest.Name, slowest.NsPerOp))

	return sb.String()
}

// GenerateComparisonTable creates a comparison table for similar benchmarks.
func GenerateComparisonTable(results []BenchmarkResult, operation string) string {
	var sb strings.Builder

	// Filter results by operation
	var filtered []BenchmarkResult
	for _, r := range results {
		if strings.Contains(r.Name, operation) {
			filtered = append(filtered, r)
		}
	}

	if len(filtered) == 0 {
		return ""
	}

	sb.WriteString(fmt.Sprintf("## %s Comparison\n\n", operation))
	sb.WriteString("| Implementation | ns/op | Relative Speed |\n")
	sb.WriteString("|----------------|-------|----------------|\n")

	// Find fastest for relative comparison
	var fastest float64
	for i, r := range filtered {
		if i == 0 || r.NsPerOp < fastest {
			fastest = r.NsPerOp
		}
	}

	// Sort by speed (fastest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].NsPerOp < filtered[j].NsPerOp
	})

	for _, r := range filtered {
		relative := r.NsPerOp / fastest
		sb.WriteString(fmt.Sprintf("| %s | %.2f | %.2fx |\n", r.Name, r.NsPerOp, relative))
	}
	sb.WriteString("\n")

	return sb.String()
}
