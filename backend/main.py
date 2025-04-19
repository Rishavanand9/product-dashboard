import pandas as pd
import time
import uuid
import os
import stat
import tempfile
import datetime
import random
from fastapi import FastAPI, UploadFile, File, BackgroundTasks
from fastapi.responses import FileResponse, JSONResponse
from typing import Dict, List, Optional
from fastapi.middleware.cors import CORSMiddleware
import undetected_chromedriver as uc
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException, NoSuchElementException

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Adjust this to specify allowed origins
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Global variables to track job status
active_jobs: Dict[str, Dict] = {}

driver_executable_path = r"D:\FinalProject\product-dashboard\backend\extras\chrome-win64\chrome-win64\chrome.exe"
st = os.stat(driver_executable_path)
os.chmod(driver_executable_path, st.st_mode | stat.S_IEXEC)

def random_sleep(min_seconds=2, max_seconds=5):
    """Random sleep to mimic human behavior and avoid bot detection"""
    sleep_time = random.uniform(min_seconds, max_seconds)
    time.sleep(sleep_time)
    return sleep_time

def get_product_info_using_selenium(item_name: str, retry_count: int = 0):
    """Get detailed product information using Selenium with retry mechanism"""
    max_retries = 3
    product_info = {}
    driver = None
    
    try:
        print(f"[{datetime.datetime.now()}] Searching Amazon for product: {item_name}")
        
        # Configure Chrome options
        options = uc.ChromeOptions()
        options.add_argument('--headless')
        options.add_argument('--no-sandbox')
        options.add_argument('--disable-dev-shm-usage')
        
        # Initialize undetected chromedriver
        driver = uc.Chrome(options=options, browser_executable_path=driver_executable_path)
        
        # Add random user agent through undetected_chromedriver config
        
        # Go to Amazon.in
        driver.get("https://www.amazon.in")
        
        random_sleep(3, 6)
        
        # Accept cookies if present
        try:
            cookie_button = WebDriverWait(driver, 5).until(
                EC.element_to_be_clickable((By.ID, "sp-cc-accept"))
            )
            cookie_button.click()
            random_sleep(1, 3)
        except TimeoutException:
            # Cookie dialog might not appear
            pass
        
        # Search for the product
        search_box = WebDriverWait(driver, 10).until(
            EC.presence_of_element_located((By.ID, "twotabsearchtextbox"))
        )
        search_box.clear()
        # Type like a human - letter by letter with random delays
        for char in item_name:
            search_box.send_keys(char)
            time.sleep(random.uniform(0.05, 0.2))
        
        search_button = WebDriverWait(driver, 10).until(
            EC.element_to_be_clickable((By.ID, "nav-search-submit-button"))
        )
        search_button.click()
        
        random_sleep(3, 7)
        
        # Click on the first search result
        try:
            first_result = WebDriverWait(driver, 10).until(
                EC.element_to_be_clickable((By.CSS_SELECTOR, "div.s-result-item[data-component-type='s-search-result'] img"))
            )
            first_result.click()
            
            # Switch to the new tab if opened
            if len(driver.window_handles) > 1:
                driver.switch_to.window(driver.window_handles[1])
                
            random_sleep(4, 8)
            
            # Extract product information
            product_info = extract_product_details(driver)
            
            print(f"[{datetime.datetime.now()}] Successfully scraped details for: {item_name}")
            
        except Exception as e:
            print(f"[{datetime.datetime.now()}] Error finding or clicking first result: {str(e)}")
            product_info["error"] = f"Failed to find product results: {str(e)}"
            
    except Exception as e:
        error_msg = f"Selenium error: {str(e)}"
        print(f"[{datetime.datetime.now()}] ERROR: {error_msg} for product: {item_name}")
        product_info["error"] = error_msg
        
        # Retry logic
        if retry_count < max_retries:
            retry_delay = (retry_count + 1) * 5  # Exponential backoff
            print(f"[{datetime.datetime.now()}] Retrying in {retry_delay} seconds (attempt {retry_count + 1}/{max_retries})...")
            time.sleep(retry_delay)
            if driver:
                driver.quit()
            return get_product_info_using_selenium(item_name, retry_count + 1)
            
    finally:
        # Close the browser
        if driver:
            driver.quit()
            
    return product_info

def extract_product_details(driver):
    """Extract product details from the product page"""
    product_info = {
        "title": "",
        "description": "",
        "image": "",
        "url": driver.current_url,
        "price": "",
        "composition": "",
        "discontinued": "",
        "unspsc_code": "",
        "dimensions": "",
        "weight": "",
        "manufacturer": "",
        "asin": "",
        "model_number": "",  
        "country_of_origin": "",
        "date_first_available": "",
        "included_components": "",
        "generic_name": ""
    }
    
    # Extract title
    try:
        product_info["title"] = driver.find_element(By.ID, "productTitle").text.strip()
        if product_info["title"] == "":
            product_info["title"] = driver.find_element(By.ID, "title").text.strip()
    except NoSuchElementException:
        product_info["title"] = "NA"
    
    # Extract price
    try:
        price_element = driver.find_element(By.CSS_SELECTOR, ".a-price .a-offscreen")
        product_info["price"] = price_element.get_attribute("innerHTML").strip()
    except NoSuchElementException:
        try:
            price_element = driver.find_elements(By.CLASS_NAME, "a-price-whole")[0]
            product_info["price"] = price_element.text.strip().replace("\n.", "")
        except NoSuchElementException:
            product_info["price"] = "NA"
    
    # Extract image URL
    try:
        img_element = driver.find_element(By.ID, "landingImage")
        product_info["image"] = img_element.get_attribute("src")
    except NoSuchElementException:
        product_info["image"] = "NA"
    
    # Extract product description
    try:
        description_element = driver.find_element(By.ID, "productDescription")
        product_info["description"] = description_element.text.strip()
    except NoSuchElementException:
        try:
            description_element = driver.find_element(By.CSS_SELECTOR, "#feature-bullets .a-list-item")
            product_info["description"] = "\n".join([item.text for item in driver.find_elements(By.CSS_SELECTOR, "#feature-bullets .a-list-item")])
        except NoSuchElementException:
            product_info["description"] = "NA"
    
    # Extract technical details from product information section
    try:
        # Look for the product details table
        detail_sections = driver.find_elements(By.CSS_SELECTOR, "table.a-keyvalue")
        
        for section in detail_sections:
            rows = section.find_elements(By.CSS_SELECTOR, "tr")
            for row in rows:
                try:
                    key = row.find_element(By.CSS_SELECTOR, "th").text.strip().lower()
                    value = row.find_element(By.CSS_SELECTOR, "td").text.strip()
                    
                    if "asin" in key:
                        product_info["asin"] = value
                    elif "manufacturer" in key:
                        product_info["manufacturer"] = value
                    elif "country of origin" in key:
                        product_info["country_of_origin"] = value
                    elif "date first available" in key:
                        product_info["date_first_available"] = value
                    elif "model" in key and "number" in key:
                        product_info["model_number"] = value
                    elif "item weight" in key or "weight" in key:
                        product_info["weight"] = value
                    elif "dimension" in key:
                        product_info["dimensions"] = value
                    elif "included" in key and "component" in key:
                        product_info["included_components"] = value
                    elif "generic name" in key:
                        product_info["generic_name"] = value
                    elif "composition" in key or "ingredients" in key:
                        product_info["composition"] = value
                except NoSuchElementException:
                    continue
    
    except Exception as e:
        print(f"Error extracting technical details: {str(e)}")
    
    # Check if product is discontinued
    try:
        if "currently unavailable" in driver.page_source.lower() or "we don't know when or if this item will be back in stock" in driver.page_source.lower():
            product_info["discontinued"] = "Yes"
        else:
            product_info["discontinued"] = "No"
    except:
        product_info["discontinued"] = "NA"
        
    return product_info

def process_excel_background(temp_file_path: str, batch_size: int = 5, job_id: str = None):
    """Process Excel file in batches with regular status updates"""
    try:
        print(f"[{datetime.datetime.now()}] Starting job {job_id}: Reading Excel file")
        df = pd.read_excel(temp_file_path)
        total_products = len(df)
        results = []
        processed = 0
        
        print(f"[{datetime.datetime.now()}] Job {job_id}: Found {total_products} products to process")
        active_jobs[job_id]["total"] = total_products
        
        # Process in batches
        for i in range(0, total_products, batch_size):
            batch = df.iloc[i:i+batch_size]
            batch_num = i//batch_size + 1
            total_batches = (total_products-1)//batch_size + 1
            print(f"[{datetime.datetime.now()}] Job {job_id}: Processing batch {batch_num}/{total_batches}")
            
            for _, row in batch.iterrows():
                name = row.get("Item Name")
                if not name:
                    print(f"[{datetime.datetime.now()}] Job {job_id}: Skipping empty product name")
                    processed += 1
                    continue

                # Get product info using Selenium
                product_info = get_product_info_using_selenium(name)
                
                # Create a flattened result dictionary with all the information
                result = {
                    "Item Name": name,
                    "Title": product_info.get("title", ""),
                    "Description": product_info.get("description", ""),
                    "Composition": product_info.get("composition", ""),
                    "Price": product_info.get("price", ""),
                    "Image URL": product_info.get("image", ""),
                    "Amazon URL": product_info.get("url", ""),
                    "Is Discontinued": product_info.get("discontinued", ""),
                    "UNSPSC Code": product_info.get("unspsc_code", ""),
                    "Product Dimensions": product_info.get("dimensions", ""),
                    "Item Weight": product_info.get("weight", ""),
                    "Manufacturer": product_info.get("manufacturer", ""),
                    "ASIN": product_info.get("asin", ""),
                    "Model Number": product_info.get("model_number", ""),
                    "Country of Origin": product_info.get("country_of_origin", ""),
                    "Date First Available": product_info.get("date_first_available", ""),
                    "Included Components": product_info.get("included_components", ""),
                    "Generic Name": product_info.get("generic_name", ""),
                    "Error": product_info.get("error", "")
                }
                
                results.append(result)
                
                processed += 1
                active_jobs[job_id]["processed"] = processed
                progress_pct = (processed/total_products*100)
                print(f"[{datetime.datetime.now()}] Job {job_id}: Progress {processed}/{total_products} ({progress_pct:.1f}%)")
                
                # Add a longer delay between individual searches to avoid detection
                wait_time = random.uniform(10, 20)
                print(f"[{datetime.datetime.now()}] Job {job_id}: Waiting {wait_time:.1f} seconds before next item")
                time.sleep(wait_time)
            
            # Add an even longer delay between batches to respect rate limits
            if i + batch_size < total_products:
                pause_time = random.uniform(30, 60)
                print(f"[{datetime.datetime.now()}] Job {job_id}: Batch {batch_num} complete. Pausing for {pause_time:.1f} seconds before next batch")
                time.sleep(pause_time)
        
        print(f"[{datetime.datetime.now()}] Job {job_id}: All products processed. Saving results to Excel")
        output = f"output_{job_id}.xlsx"
        pd.DataFrame(results).to_excel(output, index=False)
        
        active_jobs[job_id]["status"] = "completed"
        active_jobs[job_id]["output_file"] = output
        print(f"[{datetime.datetime.now()}] Job {job_id}: Job completed successfully")
        
    except Exception as e:
        error_msg = f"Error processing Excel: {str(e)}"
        print(f"[{datetime.datetime.now()}] Job {job_id}: ERROR: {error_msg}")
        active_jobs[job_id]["status"] = "failed"
        active_jobs[job_id]["error"] = error_msg
    finally:
        # Clean up the temporary file
        try:
            os.unlink(temp_file_path)
            print(f"[{datetime.datetime.now()}] Job {job_id}: Temporary file cleaned up")
        except Exception as e:
            print(f"[{datetime.datetime.now()}] Job {job_id}: Failed to cleanup temporary file: {str(e)}")

@app.post("/upload/")
async def upload_excel(background_tasks: BackgroundTasks, file: UploadFile = File(...)):
    """Upload Excel file with product names for processing"""
    try:
        print(f"[{datetime.datetime.now()}] Received file upload: {file.filename}")
        
        # Create a temporary file to store the uploaded content
        with tempfile.NamedTemporaryFile(delete=False, suffix='.xlsx') as temp_file:
            # Write the file content to the temporary file
            content = await file.read()
            temp_file.write(content)
            temp_file_path = temp_file.name
        
        # Generate a job ID
        job_id = uuid.uuid4().hex[:8]
        
        # Initialize job status
        active_jobs[job_id] = {
            "status": "processing",
            "processed": 0,
            "total": 0,
            "start_time": datetime.datetime.now().isoformat(),
            "file_name": file.filename
        }
        
        print(f"[{datetime.datetime.now()}] New job created: {job_id} for file {file.filename}")
        
        # Start processing in background
        background_tasks.add_task(process_excel_background, temp_file_path, batch_size=5, job_id=job_id)
        
        return {
            "job_id": job_id, 
            "message": "Processing started. Use /status/{job_id} to check progress and /download/{job_id} to get results when complete"
        }
    except Exception as e:
        error_msg = f"Error handling upload: {str(e)}"
        print(f"[{datetime.datetime.now()}] ERROR: {error_msg}")
        return JSONResponse(status_code=500, content={"error": error_msg})

@app.get("/status/{job_id}")
async def get_job_status(job_id: str):
    """Check the status of a processing job"""
    if job_id not in active_jobs:
        return JSONResponse(status_code=404, content={"error": "Job not found"})
    
    job_info = active_jobs[job_id].copy()
    
    # Calculate and add progress percentage
    if job_info["total"] > 0:
        job_info["progress_percentage"] = round((job_info["processed"] / job_info["total"]) * 100, 1)
    else:
        job_info["progress_percentage"] = 0
        
    # Calculate elapsed time
    start_time = datetime.datetime.fromisoformat(job_info["start_time"])
    elapsed = (datetime.datetime.now() - start_time).total_seconds()
    job_info["elapsed_seconds"] = elapsed
    job_info["elapsed_formatted"] = str(datetime.timedelta(seconds=int(elapsed)))
    
    return job_info

@app.get("/download/{job_id}")
async def download_results(job_id: str):
    """Download the results of a completed job"""
    if job_id not in active_jobs:
        return JSONResponse(status_code=404, content={"error": "Job not found"})
    
    job = active_jobs[job_id]
    if job["status"] != "completed":
        return JSONResponse(
            status_code=400, 
            content={
                "error": f"Job is not completed. Current status: {job['status']}",
                "processed": job["processed"],
                "total": job["total"],
                "progress_percentage": round((job["processed"] / job["total"]) * 100, 1) if job["total"] > 0 else 0
            }
        )
    
    output_file = job.get("output_file")
    if not output_file or not os.path.exists(output_file):
        return JSONResponse(status_code=404, content={"error": "Output file not found"})
    
    print(f"[{datetime.datetime.now()}] Sending results file for job {job_id}: {output_file}")
    return FileResponse(output_file, filename=f"amazon_results_{job_id}.xlsx")

@app.get("/jobs")
async def list_jobs():
    """List all active and completed jobs"""
    job_summaries = {}
    for job_id, job_info in active_jobs.items():
        job_summaries[job_id] = {
            "status": job_info["status"],
            "file_name": job_info.get("file_name", "Unknown"),
            "processed": job_info["processed"],
            "total": job_info["total"],
            "progress_percentage": round((job_info["processed"] / job_info["total"]) * 100, 1) if job_info["total"] > 0 else 0,
            "start_time": job_info["start_time"]
        }
    return job_summaries