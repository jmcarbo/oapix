package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmcarbo/oapix/pkg/gen"
)

const version = "0.1.0"

func main() {
	var (
		specPath      = flag.String("spec", "", "Path to OpenAPI specification file (required)")
		outputDir     = flag.String("output", ".", "Output directory for generated code")
		packageName   = flag.String("package", "", "Package name for generated code (required)")
		clientName    = flag.String("client", "Client", "Name of the generated client struct")
		templateDir   = flag.String("templates", "", "Directory containing custom templates")
		modelPackage  = flag.String("model-package", "", "Package name for models (defaults to package name)")
		clientPackage = flag.String("client-package", "", "Package name for client (defaults to package name)")
		clientImport  = flag.String("client-import", "", "Custom import path for client packages (defaults to github.com/jmcarbo/oapix/pkg/client)")
		generateAll   = flag.Bool("all", true, "Generate both models and client")
		modelsOnly    = flag.Bool("models-only", false, "Generate only models")
		clientOnly    = flag.Bool("client-only", false, "Generate only client")
		embedClient   = flag.Bool("embed-client", false, "Copy client packages instead of importing from library")
		verbose       = flag.Bool("verbose", false, "Enable verbose output")
		showVersion   = flag.Bool("version", false, "Show version information")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate Go client code from OpenAPI specifications.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate client and models\n")
		fmt.Fprintf(os.Stderr, "  %s -spec api.yaml -package myapi -output ./myapi\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Generate only models\n")
		fmt.Fprintf(os.Stderr, "  %s -spec api.yaml -package myapi -output ./myapi -models-only\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Use custom templates\n")
		fmt.Fprintf(os.Stderr, "  %s -spec api.yaml -package myapi -output ./myapi -templates ./templates\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Use custom client import path\n")
		fmt.Fprintf(os.Stderr, "  %s -spec api.yaml -package myapi -output ./myapi -client-import github.com/myorg/myclient\n\n", os.Args[0])
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("oapix-gen version %s\n", version)
		os.Exit(0)
	}

	// Validate required flags
	if *specPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -spec flag is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *packageName == "" {
		fmt.Fprintf(os.Stderr, "Error: -package flag is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Determine what to generate
	generateModels := *generateAll || *modelsOnly
	generateClient := *generateAll || *clientOnly

	if *modelsOnly && *clientOnly {
		fmt.Fprintf(os.Stderr, "Error: -models-only and -client-only are mutually exclusive\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create absolute paths
	absSpecPath, err := filepath.Abs(*specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve spec path: %v\n", err)
		os.Exit(1)
	}

	absOutputDir, err := filepath.Abs(*outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve output directory: %v\n", err)
		os.Exit(1)
	}

	// Create configuration
	config := &gen.Config{
		SpecPath:       absSpecPath,
		OutputDir:      absOutputDir,
		PackageName:    *packageName,
		ClientName:     *clientName,
		TemplateDir:    *templateDir,
		ModelPackage:   *modelPackage,
		ClientPackage:  *clientPackage,
		ClientImport:   *clientImport,
		GenerateModels: generateModels,
		GenerateClient: generateClient,
		EmbedClient:    *embedClient,
		Verbose:        *verbose,
	}

	// Create generator
	generator, err := gen.NewGenerator(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create generator: %v\n", err)
		os.Exit(1)
	}

	// Load specification
	if *verbose {
		fmt.Printf("Loading OpenAPI specification from %s...\n", *specPath)
	}

	if err := generator.LoadSpec(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load specification: %v\n", err)
		os.Exit(1)
	}

	// Generate code
	if *verbose {
		fmt.Printf("Generating code in %s...\n", *outputDir)
	}

	if err := generator.Generate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: code generation failed: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Println("Code generation completed successfully!")
	}
}
