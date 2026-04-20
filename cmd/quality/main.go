package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
)

type task struct {
	name string
	run  func() error
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "quality checks failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	tasks, err := tasksFor(args)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if err := runTask(task); err != nil {
			return fmt.Errorf("%s: %w", task.name, err)
		}
	}

	return nil
}

func tasksFor(args []string) ([]task, error) {
	if len(args) == 0 {
		return allTasks(), nil
	}

	switch args[0] {
	case "check":
		return allTasks(), nil
	case "verify":
		return []task{
			verifyFormatTask(),
			testTask(),
			lintTask(),
		}, nil
	case "fmt":
		return []task{formatTask()}, nil
	case "test":
		return []task{testTask()}, nil
	case "lint":
		return []task{lintTask()}, nil
	case "help", "-h", "--help":
		printUsage()
		return nil, nil
	default:
		printUsage()
		return nil, errors.New("unknown subcommand")
	}
}

func allTasks() []task {
	return []task{
		formatTask(),
		testTask(),
		lintTask(),
	}
}

func formatTask() task {
	return task{
		name: "format",
		run: func() error {
			return runGoFmt("-w")
		},
	}
}

func testTask() task {
	return task{
		name: "test",
		run: func() error {
			return runCommand("go", "test", "./...")
		},
	}
}

func lintTask() task {
	return task{
		name: "lint",
		run: func() error {
			return runCommand("go", "tool", "staticcheck", "./...")
		},
	}
}

func verifyFormatTask() task {
	return task{
		name: "verify-format",
		run: func() error {
			return runGoFmt("-l")
		},
	}
}

func runTask(task task) error {
	fmt.Printf("==> %s\n", task.name)
	return task.run()
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func runGoFmt(mode string) error {
	files, err := goFiles(".")
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	args := append([]string{mode}, files...)
	if mode == "-l" {
		var output []byte

		cmd := exec.Command("gofmt", args...)
		cmd.Stderr = os.Stderr
		output, err = cmd.Output()
		if err != nil {
			return err
		}
		if len(output) > 0 {
			fmt.Fprint(os.Stderr, string(output))
			return errors.New("gofmt found files that need formatting")
		}

		return nil
	}

	return runCommand("gofmt", args...)
}

func goFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if slices.Contains([]string{".git", "vendor"}, entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: go run ./cmd/quality [check|verify|fmt|test|lint]\n")
}
