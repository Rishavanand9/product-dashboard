import React, { useState, useEffect } from 'react';
import styled from 'styled-components';
import axios from 'axios';
import { axiosConfig } from './api/config';
import JobsTable from './JobsTable';

const AppContainer = styled.div`
  max-width: 80%;
  margin: 0 auto;
  padding: 2rem;
  font-family: 'Arial', sans-serif;
`;

const Header = styled.header`
  text-align: center;
  margin-bottom: 2rem;
`;

const Title = styled.h1`
  color: #333;
  font-size: 1.8rem;
`;

const UploadSection = styled.div`
  background-color: #f8f9fa;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-bottom: 2rem;
  text-align: center;
`;

const HiddenFileInput = styled.input`
  display: none;
`;

const FileLabel = styled.label`
  display: inline-block;
  background-color: #4285f4;
  color: white;
  padding: 0.75rem 1.5rem;
  border-radius: 4px;
  cursor: pointer;
  font-size: 1rem;
  transition: background-color 0.3s;
  margin-bottom: 1rem;

  &:hover {
    background-color: #3367d6;
  }
`;

const ProcessButton = styled.button`
  background-color: #0f9d58;
  color: white;
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 1rem;
  transition: background-color 0.3s;
  margin-top: 1rem;
  width: 200px;

  &:hover {
    background-color: #0b8043;
  }

  &:disabled {
    background-color: #cccccc;
    cursor: not-allowed;
  }
`;

const DownloadButton = styled.button`
  background-color: #ea4335;
  color: white;
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 1rem;
  transition: background-color 0.3s;
  margin-top: 1rem;
  width: 200px;

  &:hover {
    background-color: #d32f2f;
  }
`;

const ResetButton = styled.button`
  background-color:rgb(87, 95, 91);
  color: white;
  margin-left: 1rem;
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 1rem;
  transition: background-color 0.3s;
  margin-top: 1rem;
  width: 200px;

  &:hover {
    background-color:rgba(35, 37, 36, 0.87);
  }

  &:disabled {
    background-color: #cccccc;
    cursor: not-allowed;
  }
`;

const StatusMessage = styled.p`
  margin-top: 1rem;
  color: ${props => (props.error ? '#d32f2f' : '#43a047')};
  font-weight: ${props => (props.error ? 'bold' : 'normal')};
`;

const ProgressContainer = styled.div`
  width: 100%;
  height: 8px;
  background-color: #e0e0e0;
  border-radius: 4px;
  margin-top: 1rem;
  overflow: hidden;
`;

const ProgressBar = styled.div`
  height: 100%;
  background-color: #4285f4;
  width: ${props => props.progress}%;
  transition: width 0.3s ease-in-out;
`;

const StatusContainer = styled.div`
  margin-top: 1rem;
  text-align: center;
`;

const StatusIcon = styled.div`
  display: inline-block;
  width: 24px;
  height: 24px;
  margin-right: 8px;
  vertical-align: middle;
  ${props => props.processing && `
    border: 3px solid rgba(66, 133, 244, 0.3);
    border-top: 3px solid #4285f4;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    
    @keyframes spin {
      0% { transform: rotate(0deg); }
      100% { transform: rotate(360deg); }
    }
  `}
  ${props => props.success && `
    background-color: #0f9d58;
    border-radius: 50%;
  `}
  ${props => props.error && `
    background-color: #ea4335;
    border-radius: 50%;
  `}
`;
// Add new styled components
const FileInfo = styled.div`
  margin-top: 1rem;
  padding: 1rem;
  background-color: #f8f9fa;
  border-radius: 4px;
  text-align: left;
`;

const ElapsedTime = styled.span`
  color: #666;
  font-size: 0.9rem;
  margin-left: 1rem;
`;

const ErrorDetails = styled.pre`
  background-color: #ffebee;
  padding: 1rem;
  border-radius: 4px;
  font-size: 0.9rem;
  overflow-x: auto;
  margin-top: 1rem;
`;

const AcceptedFormats = styled.div`
  color: #666;
  font-size: 0.9rem;
  margin-top: 0.5rem;
`;

const TabContainer = styled.div`
  display: flex;
  margin-bottom: 1rem;
  width: 100%;
`;

const Tab = styled.button`
  padding: 0.75rem 1.5rem;
  border: none;
  background-color: ${props => props.active ? '#4285f4' : '#f1f3f4'};
  color: ${props => props.active ? 'white' : '#333'};
  cursor: pointer;
  font-size: 1rem;
  border-radius: 4px 4px 0 0;
  margin-right: 0.5rem;
  
  &:hover {
    background-color: ${props => props.active ? '#4285f4' : '#e0e0e0'};
  }
`;

function App() {
  const [file, setFile] = useState(null);
  const [isProcessing, setIsProcessing] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState(false);
  const [progress, setProgress] = useState(0);
  const [downloadUrl, setDownloadUrl] = useState(null);
  const [jobStatus, setJobStatus] = useState(null);
  const [statusCheckInterval, setStatusCheckInterval] = useState(null);
  const [activeTab, setActiveTab] = useState('upload');
  const [jobs, setJobs] = useState({});

  const handleFileChange = (e) => {
    const selectedFile = e.target.files?.[0];
    if (selectedFile) {
      // Validate file type
      const validTypes = ['.csv', '.xlsx'];
      const fileExtension = selectedFile.name.toLowerCase().slice(selectedFile.name.lastIndexOf('.'));
      
      if (!validTypes.includes(fileExtension)) {
        setError(true);
        setMessage('Please select a valid Excel or CSV file');
        return;
      }

      setFile(selectedFile);
      setMessage(`File selected: ${selectedFile.name}`);
      setError(false);
      setDownloadUrl(null);
      setJobStatus(null);
    }
  };

  const processFile = async () => {
    if (!file) {
      setMessage('Please select a file first');
      setError(true);
      return;
    }

    setIsProcessing(true);
    setMessage('Uploading your file...');
    setError(false);
    setProgress(0);

    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await axios.post(
        'http://localhost:8000/upload/',
        formData,
        axiosConfig
      );
      
      setMessage('File uploaded successfully! Processing has started...');
      startStatusChecking(response.data.job_id);
    } catch (error) {
      console.error('Error uploading file:', error);
      setError(true);
      setMessage('Error uploading file. Please ensure the file is not corrupted or too large.');
      setIsProcessing(false);
    }
  };

  const startStatusChecking = (jobId) => {
    // Clear any existing interval
    if (statusCheckInterval) {
      clearInterval(statusCheckInterval);
    }

    const interval = setInterval(async () => {
      try {
        const response = await axios.get(`http://localhost:8000/status/${jobId}`);
        const status = response.data;
        setJobStatus(status);
        setProgress(status.progress_percentage);

        if (status.status === 'completed') {
          clearInterval(interval);
          setDownloadUrl(`http://localhost:8000/download/${jobId}`);
          setMessage('Processing completed! Your enhanced product data is ready for download.');
          setIsProcessing(false);
        } else if (status.status === 'failed') {
          clearInterval(interval);
          setError(true);
          setMessage(`Processing failed: ${status.error || 'Unknown error occurred'}`);
          setIsProcessing(false);
        } else {
          setMessage(`Processing your products: ${status.processed} of ${status.total} items completed`);
        }
      } catch (error) {
        console.error('Error checking job status:', error);
        clearInterval(interval);
        setError(true);
        setMessage('Lost connection to the server. Please try again.');
        setIsProcessing(false);
      }
    }, 2000);

    setStatusCheckInterval(interval);
  };

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (statusCheckInterval) {
        clearInterval(statusCheckInterval);
      }
    };
  }, [statusCheckInterval]);

  const downloadFile = () => {
    if (downloadUrl) {
      window.location.href = downloadUrl;
    }
  };

  // Function to fetch all jobs
  const fetchJobs = async () => {
    try {
      const response = await axios.get('http://localhost:8000/jobs/', axiosConfig);
      setJobs(response.data);
    } catch (error) {
      console.error('Error fetching jobs:', error);
    }
  };

  // Fetch jobs when the jobs tab is selected
  useEffect(() => {
    if (activeTab === 'jobs') {
      fetchJobs();
      const interval = setInterval(fetchJobs, 5000); // Refresh every 5 seconds
      return () => clearInterval(interval);
    }
  }, [activeTab]);

  const resetForm = () => {
    setFile(null);
    setIsProcessing(false);
    setMessage('');
    setError(false);
    setProgress(0);
    setDownloadUrl(null);
    setJobStatus(null);
    
    if (statusCheckInterval) {
      clearInterval(statusCheckInterval);
      setStatusCheckInterval(null);
    }
    
    // Reset file input
    const fileInput = document.getElementById('file-upload');
    if (fileInput) fileInput.value = '';
  };

  return (
    <AppContainer>
      <Header>
        <Title>Amazon Product Details Enhancer</Title>
      </Header>
      
      <TabContainer>
        <Tab 
          active={activeTab === 'upload'} 
          onClick={() => setActiveTab('upload')}
        >
          Upload Product File
        </Tab>
        <Tab 
          active={activeTab === 'jobs'} 
          onClick={() => setActiveTab('jobs')}
        >
          Your Jobs
        </Tab>
      </TabContainer>
      
      {activeTab === 'upload' && (
        <UploadSection>
          <HiddenFileInput 
            type="file" 
            id="file-upload" 
            accept=".csv,.xlsx" 
            onChange={handleFileChange} 
          />
          <FileLabel htmlFor="file-upload">
            Select Excel/CSV File
          </FileLabel>
          
          <AcceptedFormats>
            Accepted formats: .xlsx, .csv
          </AcceptedFormats>
          
          {file && (
            <FileInfo>
              <strong>File:</strong> {file.name}
              <br />
              <strong>Size:</strong> {(file.size / 1024 / 1024).toFixed(2)} MB
            </FileInfo>
          )}
          
          {file && !isProcessing && !downloadUrl && (
            <div>
              <ProcessButton onClick={processFile} disabled={isProcessing}>
                Enhance Product Details
              </ProcessButton>
              <ResetButton onClick={resetForm}>
                Reset
              </ResetButton>
            </div>
          )}
          
          {isProcessing && jobStatus && (
            <div>
              <ProgressContainer>
                <ProgressBar progress={progress} />
              </ProgressContainer>
              <StatusContainer>
                <StatusIcon processing />
                <StatusMessage>
                  {message}
                  <ElapsedTime>
                    Time elapsed: {jobStatus.elapsed_formatted}
                  </ElapsedTime>
                </StatusMessage>
              </StatusContainer>
            </div>
          )}
          
          {downloadUrl && (
            <div>
              <StatusContainer>
                <StatusIcon success />
                <StatusMessage>{message}</StatusMessage>
              </StatusContainer>
              <DownloadButton onClick={downloadFile}>
                Download Enhanced Product Data
              </DownloadButton>
              <ResetButton onClick={resetForm}>
                Reset
              </ResetButton>
            </div>
          )}
          
          {error && (
            <StatusContainer>
              <StatusIcon error />
              <StatusMessage error>{message}</StatusMessage>
              {jobStatus?.error && (
                <ErrorDetails>{jobStatus.error}</ErrorDetails>
              )}
            </StatusContainer>
          )}
        </UploadSection>
      )}
      
      {activeTab === 'jobs' && (
        <JobsTable jobs={jobs} onRefresh={fetchJobs} />
      )}
    </AppContainer>
  );
}

export default App;