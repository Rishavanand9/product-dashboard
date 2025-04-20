package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tealeg/xlsx/v3"
)

// ProductInfo represents the details of a product
type ProductInfo struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	Image              string `json:"image"`
	URL                string `json:"url"`
	Price              string `json:"price"`
	Composition        string `json:"composition"`
	Discontinued       string `json:"discontinued"`
	UnspscCode         string `json:"unspsc_code"`
	Dimensions         string `json:"dimensions"`
	Weight             string `json:"weight"`
	Manufacturer       string `json:"manufacturer"`
	ASIN               string `json:"asin"`
	ModelNumber        string `json:"model_number"`
	CountryOfOrigin    string `json:"country_of_origin"`
	DateFirstAvailable string `json:"date_first_available"`
	IncludedComponents string `json:"included_components"`
	GenericName        string `json:"generic_name"`
	ProductDetails     string `json:"product_details_as_on_amazon.in"`
	Error              string `json:"error"`
}

// ResultRow represents a row in the results file
type ResultRow struct {
	SrNo               string
	ItemCode           string
	ItemName           string
	Title              string
	CompositionAmazon  string
	Price              string
	ProductDetails     string
	ImageURL           string
	AmazonURL          string
	IsDiscontinued     string
	UnspscCode         string
	ProductDimensions  string
	ItemWeight         string
	Manufacturer       string
	ASIN               string
	ModelNumber        string
	CountryOfOrigin    string
	DateFirstAvailable string
	IncludedComponents string
	GenericName        string
	Error              string
}

// JobInfo represents the status and information of a processing job
type JobInfo struct {
	Status            string    `json:"status"`
	Processed         int       `json:"processed"`
	Total             int       `json:"total"`
	StartTime         time.Time `json:"start_time"`
	FileName          string    `json:"file_name"`
	OutputFile        string    `json:"output_file,omitempty"`
	Error             string    `json:"error,omitempty"`
	ProgressPercentage float64  `json:"progress_percentage,omitempty"`
	ElapsedSeconds    float64   `json:"elapsed_seconds,omitempty"`
	ElapsedFormatted  string    `json:"elapsed_formatted,omitempty"`
}

var (
	activeJobs     = make(map[string]*JobInfo)
	activeJobsLock sync.RWMutex
	logger         *log.Logger
)

func init() {
	// Configure logging
	logDir := "logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.MkdirAll(logDir, 0755)
	}
	
	// Create log file
	currentTime := time.Now().Format("20060102")
	logFile, err := os.OpenFile(
		filepath.Join(logDir, fmt.Sprintf("amazon_scraper_%s.log", currentTime)),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	
	// Create multi writer to write logs to both file and console
	multiWriter := io.MultiWriter(logFile, os.Stdout)
	logger = log.New(multiWriter, "", log.LstdFlags)
	
	// Ensure output directory exists
	outputDir := filepath.Join(".", "output_files")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		os.MkdirAll(outputDir, 0755)
	}
}

// randomSleep adds a random delay between operations
func randomSleep(minSeconds, maxSeconds float64) float64 {
	sleepTime := minSeconds + rand.Float64()*(maxSeconds-minSeconds)
	time.Sleep(time.Duration(sleepTime * float64(time.Second)))
	return sleepTime
}

// getProductInfoUsingChromeDp scrapes Amazon product info using ChromeDP
func getProductInfoUsingChromeDp(itemName string, retryCount int) ProductInfo {
	maxRetries := 3
	productInfo := ProductInfo{}
	
	logger.Printf("Searching Amazon for product: %s", itemName)
	
	// Set up Chrome options
	options := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
	}
	
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer cancel()
	
	// Create Chrome instance
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(logger.Printf),
	)
	defer cancel()
	
	// Set a timeout
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	var price, title, imageURL, description, detailBullets string
	var discontinued bool
	
	err := chromedp.Run(ctx,
		// Navigate to Amazon
		chromedp.Navigate("https://www.amazon.in"),
		
		// Wait for page to load
		chromedp.WaitVisible("#twotabsearchtextbox", chromedp.ByID),
		
		// Try to accept cookies if dialog appears
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Try to click the accept cookies button
			err := chromedp.Click("#sp-cc-accept", chromedp.ByID).Do(ctx)
			// Ignore errors since the cookie dialog might not appear
			if err != nil {
				return err	
			} else {
				return nil
			}
		}),
		
		// Type in search box (simulate human typing)
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, char := range itemName {
				err := chromedp.SendKeys("#twotabsearchtextbox", string(char), chromedp.ByID).Do(ctx)
				if err != nil {
					return err
				}
				time.Sleep(time.Duration(50+rand.Intn(150)) * time.Millisecond)
			}
			return nil
		}),
		
		// Click search button
		chromedp.Click("#nav-search-submit-button", chromedp.ByID),
		
		// Wait for search results
		chromedp.WaitVisible("div.s-result-item[data-component-type='s-search-result']", chromedp.ByQuery),
		
		// Random pause
		chromedp.ActionFunc(func(ctx context.Context) error {
			randomSleep(3, 7)
			return nil
		}),
		
		// Click on first result
		chromedp.Click("div.s-result-item[data-component-type='s-search-result'] img", chromedp.ByQuery),
		
		// Wait for product page to load
		chromedp.WaitVisible("#productTitle", chromedp.ByID),
		
		// Random pause
		chromedp.ActionFunc(func(ctx context.Context) error {
			randomSleep(4, 8)
			return nil
		}),
		
		// Get product title
		chromedp.Text("#productTitle", &title, chromedp.ByID),
		
		// Get product price (try different selectors)
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			err = chromedp.Text(".a-price .a-offscreen", &price, chromedp.ByQuery).Do(ctx)
			if err != nil {
				err = chromedp.Text(".a-price-whole", &price, chromedp.ByQuery).Do(ctx)
				if err != nil {
					price = "NA"
				}
			}
			return nil
		}),
		
		// Get product image
		chromedp.AttributeValue("#landingImage", "src", &imageURL, nil, chromedp.ByID),
		
		// Get product description
		chromedp.ActionFunc(func(ctx context.Context) error {
			err := chromedp.Text("#productDescription", &description, chromedp.ByID).Do(ctx)
			if err != nil {
				// Try feature bullets instead
				desc := []string{}
				var nodes []*cdp.Node
				if err := chromedp.Nodes("#feature-bullets .a-list-item", &nodes, chromedp.ByQueryAll).Do(ctx); err == nil {
					for _, node := range nodes {
						var text string
						if err := chromedp.Text(node, &text).Do(ctx); err == nil {
							desc = append(desc, text)
						}
					}
					description = strings.Join(desc, "\n")
				} else {
					description = "NA"
				}
			}
			return nil
		}),
		
		// Get product detail bullets
		chromedp.Text("#detailBullets_feature_div", &detailBullets, chromedp.ByID),
		
		// Check if product is discontinued
		chromedp.ActionFunc(func(ctx context.Context) error {
			var pageText string
			chromedp.OuterHTML("body", &pageText, chromedp.ByQuery).Do(ctx)
			pageTextLower := strings.ToLower(pageText)
			discontinued = strings.Contains(pageTextLower, "currently unavailable") || 
				strings.Contains(pageTextLower, "we don't know when or if this item will be back in stock")
			return nil
		}),
		
		// Get product URL
		chromedp.ActionFunc(func(ctx context.Context) error {
			var locationURL string
			chromedp.Location(&locationURL).Do(ctx)
			productInfo.URL = locationURL
			return nil
		}),
	)
	
	if err != nil {
		errorMsg := fmt.Sprintf("ChromeDP error: %v", err)
		logger.Printf("%s for product: %s", errorMsg, itemName)
		productInfo.Error = errorMsg
		
		// Retry logic
		if retryCount < maxRetries {
			retryDelay := (retryCount + 1) * 5 // Exponential backoff
			logger.Printf("Retrying in %d seconds (attempt %d/%d)...", retryDelay, retryCount+1, maxRetries)
			time.Sleep(time.Duration(retryDelay) * time.Second)
			return getProductInfoUsingChromeDp(itemName, retryCount+1)
		}
		
		return productInfo
	}
	
	// Process the data we collected
	productInfo.Title = strings.TrimSpace(title)
	productInfo.Price = strings.TrimSpace(price)
	productInfo.Image = imageURL
	productInfo.Description = strings.TrimSpace(description)
	
	if discontinued {
		productInfo.Discontinued = "Yes"
	} else {
		productInfo.Discontinued = "No"
	}
	
	// Parse technical details from detail bullets
	extractProductDetails(&productInfo, detailBullets)
	
	return productInfo
}

// extractProductDetails parses the product detail text to extract technical details
func extractProductDetails(productInfo *ProductInfo, detailText string) {
	productInfo.ProductDetails = strings.TrimSpace(detailText)
	
	// Simple parsing of common Amazon detail format "key : value"
	lines := strings.Split(detailText, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.ToLower(strings.TrimSpace(parts[0]))
			value := strings.TrimSpace(parts[1])
			
			// Set values based on keys
			switch {
			case strings.Contains(key, "asin"):
				productInfo.ASIN = value
			case strings.Contains(key, "manufacturer"):
				productInfo.Manufacturer = value
			case strings.Contains(key, "country of origin"):
				productInfo.CountryOfOrigin = value
			case strings.Contains(key, "date first available"):
				productInfo.DateFirstAvailable = value
			case strings.Contains(key, "model") && strings.Contains(key, "number"):
				productInfo.ModelNumber = value
			case strings.Contains(key, "weight"):
				productInfo.Weight = value
			case strings.Contains(key, "dimension"):
				productInfo.Dimensions = value
			case strings.Contains(key, "included") && strings.Contains(key, "component"):
				productInfo.IncludedComponents = value
			case strings.Contains(key, "generic name"):
				productInfo.GenericName = value
			case strings.Contains(key, "composition") || strings.Contains(key, "ingredients"):
				productInfo.Composition = value
			}
		}
	}
}

// processFile processes a file of product names in batches
func processFile(filePath, fileType, jobID string) {
	defer func() {
		// Clean up the temporary file
		os.Remove(filePath)
		logger.Printf("Job %s: Temporary file cleaned up", jobID)
	}()
	
	var products [][]string
	var headers []string
	var err error
	
	logger.Printf("Starting job %s: Reading file %s", jobID, filePath)
	
	// Read the file based on type
	if fileType == "xlsx" {
		products, headers, err = readExcelFile(filePath)
	} else if fileType == "csv" {
		products, headers, err = readCSVFile(filePath)
	}
	
	if err != nil {
		errorMsg := fmt.Sprintf("Error reading file: %v", err)
		logger.Printf(errorMsg)
		updateJobStatus(jobID, "failed", 0, 0, "", errorMsg)
		return
	}
	
	totalProducts := len(products)
	updateJobTotalCount(jobID, totalProducts)
	logger.Printf("Job %s: Found %d products to process", jobID, totalProducts)
	
	// Prepare results slice
	results := make([]ResultRow, 0, totalProducts)
	
	// Find column indices
	srnoIdx := findColumnIndex(headers, "SrNo")
	itemCodeIdx := findColumnIndex(headers, "Item Code")
	nameIdx := findColumnIndex(headers, "Item Name")
	
	// Process in batches
	batchSize := 5
	processed := 0
	
	for i := 0; i < totalProducts; i += batchSize {
		end := i + batchSize
		if end > totalProducts {
			end = totalProducts
		}
		batch := products[i:end]
		batchNum := i/batchSize + 1
		totalBatches := (totalProducts-1)/batchSize + 1
		logger.Printf("Job %s: Processing batch %d/%d", jobID, batchNum, totalBatches)
		
		for _, row := range batch {
			// Extract product info
			srno := "NA"
			if srnoIdx >= 0 && srnoIdx < len(row) {
				srno = row[srnoIdx]
			}
			
			itemCode := "NA"
			if itemCodeIdx >= 0 && itemCodeIdx < len(row) {
				itemCode = row[itemCodeIdx]
			}
			
			name := ""
			if nameIdx >= 0 && nameIdx < len(row) {
				name = row[nameIdx]
			}
			
			if name == "" {
				logger.Printf("Job %s: Skipping empty product name", jobID)
				processed++
				updateJobProgress(jobID, processed)
				continue
			}
			
			// Get product info
			productInfo := getProductInfoUsingChromeDp(name, 0)
			
			// Create result row
			result := ResultRow{
				SrNo:               srno,
				ItemCode:           itemCode,
				ItemName:           name,
				Title:              productInfo.Title,
				CompositionAmazon:  productInfo.Description,
				Price:              productInfo.Price,
				ProductDetails:     productInfo.ProductDetails,
				ImageURL:           productInfo.Image,
				AmazonURL:          productInfo.URL,
				IsDiscontinued:     productInfo.Discontinued,
				UnspscCode:         productInfo.UnspscCode,
				ProductDimensions:  productInfo.Dimensions,
				ItemWeight:         productInfo.Weight,
				Manufacturer:       productInfo.Manufacturer,
				ASIN:               productInfo.ASIN,
				ModelNumber:        productInfo.ModelNumber,
				CountryOfOrigin:    productInfo.CountryOfOrigin,
				DateFirstAvailable: productInfo.DateFirstAvailable,
				IncludedComponents: productInfo.IncludedComponents,
				GenericName:        productInfo.GenericName,
				Error:              productInfo.Error,
			}
			
			results = append(results, result)
			
			processed++
			updateJobProgress(jobID, processed)
			progressPct := float64(processed) / float64(totalProducts) * 100
			logger.Printf("Job %s: Progress %d/%d (%.1f%%)", jobID, processed, totalProducts, progressPct)
			
			// Add delay between individual searches
			waitTime := randomSleep(10, 20)
			logger.Printf("Job %s: Waiting %.1f seconds before next item", jobID, waitTime)
		}
		
		// Add longer delay between batches
		if i+batchSize < totalProducts {
			pauseTime := randomSleep(30, 60)
			logger.Printf("Job %s: Batch %d complete. Pausing for %.1f seconds before next batch", 
			    jobID, batchNum, pauseTime)
		}
	}
	
	logger.Printf("Job %s: All products processed. Saving results", jobID)
	
	var outputPath string
	if fileType == "xlsx" {
		outputPath = filepath.Join("output_files", fmt.Sprintf("output_%s.xlsx", jobID))
		err = writeExcelFile(outputPath, results)
	} else {
		outputPath = filepath.Join("output_files", fmt.Sprintf("output_%s.csv", jobID))
		err = writeCSVFile(outputPath, results)
	}
	
	if err != nil {
		errorMsg := fmt.Sprintf("Error saving results: %v", err)
		logger.Printf(errorMsg)
		updateJobStatus(jobID, "failed", processed, totalProducts, "", errorMsg)
		return
	}
	
	updateJobStatus(jobID, "completed", processed, totalProducts, outputPath, "")
	logger.Printf("Job %s: Job completed successfully", jobID)
}

// Helper functions for file handling

func readExcelFile(filePath string) ([][]string, []string, error) {
	wb, err := xlsx.OpenFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	
	if len(wb.Sheets) == 0 {
		return nil, nil, fmt.Errorf("no sheets in Excel file")
	}
	
	sheet := wb.Sheets[0]
	rows := make([][]string, 0)
	headers := make([]string, 0)
	
	// Read header row
	if sheet.MaxRow > 0 {
		headerRow, err := sheet.Row(0)
		if err != nil {
			return nil, nil, err
		}
		
		for i := 0; i < sheet.MaxCol; i++ {
			cell := headerRow.GetCell(i)
			if cell != nil {
				headers = append(headers, "")
				continue
			}
			headers = append(headers, cell.String())
		}
	}
	
	// Read data rows
	for r := 1; r < sheet.MaxRow; r++ {
		row, err := sheet.Row(r)
		if err != nil {
			continue
		}
		
		rowData := make([]string, 0, sheet.MaxCol)
		for c := 0; c < sheet.MaxCol; c++ {
			cell := row.GetCell(c)
			if cell != nil {
				rowData = append(rowData, "")
				continue
			}
			rowData = append(rowData, cell.String())
		}
		rows = append(rows, rowData)
	}
	
	return rows, headers, nil
}

func readCSVFile(filePath string) ([][]string, []string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	
	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return nil, nil, err
	}
	
	// Read data rows
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	
	return rows, headers, nil
}

func writeExcelFile(filePath string, results []ResultRow) error {
	wb := xlsx.NewFile()
	sheet, err := wb.AddSheet("Results")
	if err != nil {
		return err
	}
	
	// Add header row
	headerRow := sheet.AddRow()
	headerRow.AddCell().SetString("SrNo")
	headerRow.AddCell().SetString("Item Code")
	headerRow.AddCell().SetString("Item Name")
	headerRow.AddCell().SetString("Title")
	headerRow.AddCell().SetString("Composition_on_amazon.in")
	headerRow.AddCell().SetString("Price")
	headerRow.AddCell().SetString("Product Details as on amazon.in")
	headerRow.AddCell().SetString("Image URL")
	headerRow.AddCell().SetString("Amazon URL")
	headerRow.AddCell().SetString("Is Discontinued")
	headerRow.AddCell().SetString("UNSPSC Code")
	headerRow.AddCell().SetString("Product Dimensions")
	headerRow.AddCell().SetString("Item Weight")
	headerRow.AddCell().SetString("Manufacturer")
	headerRow.AddCell().SetString("ASIN")
	headerRow.AddCell().SetString("Model Number")
	headerRow.AddCell().SetString("Country of Origin")
	headerRow.AddCell().SetString("Date First Available")
	headerRow.AddCell().SetString("Included Components")
	headerRow.AddCell().SetString("Generic Name")
	headerRow.AddCell().SetString("Error")
	
	// Add data rows
	for _, result := range results {
		row := sheet.AddRow()
		row.AddCell().SetString(result.SrNo)
		row.AddCell().SetString(result.ItemCode)
		row.AddCell().SetString(result.ItemName)
		row.AddCell().SetString(result.Title)
		row.AddCell().SetString(result.CompositionAmazon)
		row.AddCell().SetString(result.Price)
		row.AddCell().SetString(result.ProductDetails)
		row.AddCell().SetString(result.ImageURL)
		row.AddCell().SetString(result.AmazonURL)
		row.AddCell().SetString(result.IsDiscontinued)
		row.AddCell().SetString(result.UnspscCode)
		row.AddCell().SetString(result.ProductDimensions)
		row.AddCell().SetString(result.ItemWeight)
		row.AddCell().SetString(result.Manufacturer)
		row.AddCell().SetString(result.ASIN)
		row.AddCell().SetString(result.ModelNumber)
		row.AddCell().SetString(result.CountryOfOrigin)
		row.AddCell().SetString(result.DateFirstAvailable)
		row.AddCell().SetString(result.IncludedComponents)
		row.AddCell().SetString(result.GenericName)
		row.AddCell().SetString(result.Error)
	}
	
	return wb.Save(filePath)
}

func writeCSVFile(filePath string, results []ResultRow) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write header row
	err = writer.Write([]string{
		"SrNo", "Item Code", "Item Name", "Title", "Composition_on_amazon.in", "Price",
		"Product Details as on amazon.in", "Image URL", "Amazon URL", "Is Discontinued",
		"UNSPSC Code", "Product Dimensions", "Item Weight", "Manufacturer", "ASIN",
		"Model Number", "Country of Origin", "Date First Available", "Included Components",
		"Generic Name", "Error",
	})
	if err != nil {
		return err
	}
	
	// Write data rows
	for _, result := range results {
		err = writer.Write([]string{
			result.SrNo, result.ItemCode, result.ItemName, result.Title,
			result.CompositionAmazon, result.Price, result.ProductDetails,
			result.ImageURL, result.AmazonURL, result.IsDiscontinued,
			result.UnspscCode, result.ProductDimensions, result.ItemWeight,
			result.Manufacturer, result.ASIN, result.ModelNumber,
			result.CountryOfOrigin, result.DateFirstAvailable,
			result.IncludedComponents, result.GenericName, result.Error,
		})
		if err != nil {
			return err
		}
	}
	
	return nil
}

// Helper function to find column index by name
func findColumnIndex(headers []string, columnName string) int {
	for i, header := range headers {
		if strings.EqualFold(header, columnName) {
			return i
		}
	}
	return -1
}

// Job management functions

func updateJobStatus(jobID, status string, processed, total int, outputFile, errorMsg string) {
	activeJobsLock.Lock()
	defer activeJobsLock.Unlock()
	
	if job, ok := activeJobs[jobID]; ok {
		job.Status = status
		job.Processed = processed
		job.Total = total
		if outputFile != "" {
			job.OutputFile = outputFile
		}
		if errorMsg != "" {
			job.Error = errorMsg
		}
	}
}

func updateJobProgress(jobID string, processed int) {
	activeJobsLock.Lock()
	defer activeJobsLock.Unlock()
	
	if job, ok := activeJobs[jobID]; ok {
		job.Processed = processed
	}
}

func updateJobTotalCount(jobID string, total int) {
	activeJobsLock.Lock()
	defer activeJobsLock.Unlock()
	
	if job, ok := activeJobs[jobID]; ok {
		job.Total = total
	}
}

func getJobInfo(jobID string) (*JobInfo, bool) {
	activeJobsLock.RLock()
	defer activeJobsLock.RUnlock()
	
	job, ok := activeJobs[jobID]
	if !ok {
		return nil, false
	}
	
	// Create a copy to modify
	jobCopy := *job
	
	// Calculate progress percentage
	if jobCopy.Total > 0 {
		jobCopy.ProgressPercentage = float64(jobCopy.Processed) / float64(jobCopy.Total) * 100
	}
	
	// Calculate elapsed time
	elapsed := time.Since(jobCopy.StartTime).Seconds()
	jobCopy.ElapsedSeconds = elapsed
	jobCopy.ElapsedFormatted = formatDuration(elapsed)
	
	return &jobCopy, true
}

func formatDuration(seconds float64) string {
	duration := time.Duration(seconds) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60
	
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// API Handlers

func uploadFileHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Error getting file: %v", err)})
		return
	}
	
	// Check file type
	fileName := file.Filename
	var fileType string
	if strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
		fileType = "xlsx"
	} else if strings.HasSuffix(strings.ToLower(fileName), ".csv") {
		fileType = "csv"
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file type"})
		return
	}
	
	// Generate a job ID
	jobID := uuid.New().String()[:8]
	
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "upload-*"+filepath.Ext(fileName))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creating temp file: %v", err)})
		return
	}
	tempFilePath := tempFile.Name()
	tempFile.Close()
	
	// Save the uploaded file to the temporary file
	if err := c.SaveUploadedFile(file, tempFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error saving file: %v", err)})
		return
	}
	
	// Initialize job status
	activeJobsLock.Lock()
	activeJobs[jobID] = &JobInfo{
		Status:    "processing",
		Processed: 0,
		Total:     0,
		StartTime: time.Now(),
		FileName:  fileName,
	}
	activeJobsLock.Unlock()
	
	logger.Printf("New job created: %s for file %s", jobID, fileName)
	
	// Start processing in background
	go processFile(tempFilePath, fileType, jobID)
	
	c.JSON(http.StatusOK, gin.H{
		"job_id":  jobID,
		"message": "Processing started. Use /status/{job_id} to check progress and /download/{job_id} to get results when complete",
	})
}

func getJobStatusHandler(c *gin.Context) {
	jobID := c.Param("job_id")
	
	jobInfo, ok := getJobInfo(jobID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}
	
	c.JSON(http.StatusOK, jobInfo)
}

func downloadResultsHandler(c *gin.Context) {
	jobID := c.Param("job_id")
	
	jobInfo, ok := getJobInfo(jobID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}
	
	if jobInfo.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Job is not completed. Current status: %s", jobInfo.Status),
			"processed": jobInfo.Processed,
			"total": jobInfo.Total,
			"progress_percentage": jobInfo.ProgressPercentage,
		})
		return
	}
	
	outputFile := jobInfo.OutputFile
	if outputFile == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Output file not found"})
		return
	}
	
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
        	c.JSON(http.StatusNotFound, gin.H{"error": "Output file not found"})
        	return
    	}
	
	logger.Printf("Sending results file for job %s: %s", jobID, outputFile)
	
	// Get file extension
	fileExt := filepath.Ext(outputFile)
	fileName := fmt.Sprintf("amazon_results_%s%s", jobID, fileExt)
	
	c.FileAttachment(outputFile, fileName)
}

func listJobsHandler(c *gin.Context) {
	activeJobsLock.RLock()
	defer activeJobsLock.RUnlock()
	
	jobSummaries := make(map[string]gin.H)
	for jobID, jobInfo := range activeJobs {
		progressPct := 0.0
		if jobInfo.Total > 0 {
			progressPct = float64(jobInfo.Processed) / float64(jobInfo.Total) * 100
		}
		
		jobSummaries[jobID] = gin.H{
			"status": jobInfo.Status,
			"file_name": jobInfo.FileName,
			"processed": jobInfo.Processed,
			"total": jobInfo.Total,
			"progress_percentage": progressPct,
			"start_time": jobInfo.StartTime,
		}
	}
	
	c.JSON(http.StatusOK, jobSummaries)
}

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
	
	// Create Gin router
	router := gin.Default()
	
	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: true,
	}))
	
	// Setup routes
	router.POST("/upload", uploadFileHandler)
	router.GET("/status/:job_id", getJobStatusHandler)
	router.GET("/download/:job_id", downloadResultsHandler)
	router.GET("/jobs", listJobsHandler)
	
	// Start server
	logger.Println("Starting Amazon Scraper server on :8080")
	if err := router.Run(":8080"); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
