import pandas as pd
import time
import uuid
import os
import tempfile
import json
import re
import datetime
import google.generativeai as genai
from fastapi import FastAPI, UploadFile, File, BackgroundTasks
from fastapi.responses import FileResponse, JSONResponse
from typing import Dict, List, Optional
from fastapi.middleware.cors import CORSMiddleware

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Adjust this to specify allowed origins
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# Configure Gemini API
API_KEY = ""  # Replace with your actual Gemini API key
genai.configure(api_key=API_KEY)

# Global variables to track job status
active_jobs: Dict[str, Dict] = {}

def get_product_info_from_gemini(item_name: str, retry_count: int = 0):
    """Get detailed product information from Gemini API with retry mechanism"""
    max_retries = 3
    try:
        print(f"[{datetime.datetime.now()}] Requesting Gemini API for product: {item_name}")
        
        # Enhanced prompt to get specific product details
        prompt = f"""
        Search for the product "{item_name}" on Amazon.in and provide detailed information in JSON format.
        Include the following fields:
        
        1. Basic Information:
           - title: Product title
           - description: Brief product description
           - image: Image URL if available
           - url: Amazon product URL
           - price: Current price
           - composition: Product composition or ingredients
        
        2. Technical Details (if available):
           - discontinued: Is product discontinued (Yes/No)
           - unspsc_code: UNSPSC Code and category
           - dimensions: Product dimensions
           - weight: Item weight
           - manufacturer: Manufacturer name
           - asin: Amazon ASIN
           - model_number: Item model number
           - country_of_origin: Country of origin
           - date_first_available: When product was first available
           - included_components: What's included in the package
           - generic_name: Generic product name
        
        Format the response as valid JSON with these exact keys. If any information is not available, use empty string or "N/A".
        """
        
        # Make request to Gemini
        model = genai.GenerativeModel('gemini-1.5-pro')
        response = model.generate_content(prompt)
        
        # Parse the response
        try:
            # Find JSON-like content in the response
            json_match = re.search(r'```json\s*(.*?)\s*```', response.text, re.DOTALL)
            if json_match:
                json_text = json_match.group(1)
            else:
                json_text = response.text
                
            # Clean up and parse the JSON
            product_info = json.loads(json_text)
            
            print(f"[{datetime.datetime.now()}] Successfully processed: {item_name}")
            return product_info
            
        except Exception as json_error:
            error_msg = f"Failed to parse Gemini response: {str(json_error)}"
            print(f"[{datetime.datetime.now()}] ERROR: {error_msg} for product: {item_name}")
            return {
                "error": error_msg,
                "raw_response": response.text
            }
            
    except Exception as e:
        error_msg = f"Gemini API error: {str(e)}"
        print(f"[{datetime.datetime.now()}] ERROR: {error_msg} for product: {item_name}")
        
        # Retry logic
        if retry_count < max_retries:
            retry_delay = (retry_count + 1) * 5  # Exponential backoff
            print(f"[{datetime.datetime.now()}] Retrying in {retry_delay} seconds (attempt {retry_count + 1}/{max_retries})...")
            time.sleep(retry_delay)
            return get_product_info_from_gemini(item_name, retry_count + 1)
        else:
            return {"error": error_msg}

def process_excel_background(temp_file_path: str, batch_size: int = 10, job_id: str = None):
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

                # Get product info from Gemini API
                product_info = get_product_info_from_gemini(name)
                
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
                
                # Add a short delay between individual API calls
                time.sleep(1)
            
            # Add a longer delay between batches to respect API rate limits
            if i + batch_size < total_products:
                pause_time = 5
                print(f"[{datetime.datetime.now()}] Job {job_id}: Batch {batch_num} complete. Pausing for {pause_time} seconds before next batch")
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
        background_tasks.add_task(process_excel_background, temp_file_path, batch_size=10, job_id=job_id)
        
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