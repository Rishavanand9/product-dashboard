
import React, { useState } from 'react';
import styled from 'styled-components';
import axios from 'axios';

const AppContainer = styled.div`
  max-width: 800px;
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

const StatusMessage = styled.p`
  margin-top: 1rem;
  color: ${props => (props.error ? '#d32f2f' : '#43a047')};
  font-weight: ${props => (props.error ? 'bold' : 'normal')};
`;

const FileName = styled.div`
  margin-top: 1rem;
  padding: 0.5rem;
  background-color: #e9ecef;
  border-radius: 4px;
  display: inline-block;
  min-width: 200px;
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

function App() {
  const [file, setFile] = useState(null);
  const [isProcessing, setIsProcessing] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState(false);
  const [progress, setProgress] = useState(0);
  const [downloadUrl, setDownloadUrl] = useState(null);

  const handleFileChange = (e) => {
    const selectedFile = e.target.files[0];
    if (selectedFile) {
      setFile(selectedFile);
      setMessage(`File selected: ${selectedFile.name}`);
      setError(false);
      setDownloadUrl(null);
    }
  };

  const processFile = async () => {
    if (!file) {
      setMessage('Please select a file first');
      setError(true);
      return;
    }

    setIsProcessing(true);
    setMessage('Processing your file...');
    setError(false);
    setProgress(0);

    // Create a FormData object to send the file
    const formData = new FormData();
    formData.append('file', file);

    try {
      // Simulating progress updates
      const progressInterval = setInterval(() => {
        setProgress(prev => {
          const newProgress = prev + Math.random() * 10;
          return newProgress >= 95 ? 95 : newProgress;
        });
      }, 500);

      // Replace with your actual API endpoint
      const response = await axios.post('https://your-api-endpoint.com/process', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });
      
      // Clear the interval and set progress to 100%
      clearInterval(progressInterval);
      setProgress(100);
      
      // In a real application, the API would return a URL or blob for download
      // For demo purposes, we'll create a fake download URL
      const responseData = response.data || {};
      
      // Simulating a download URL from the API response
      // In a real app, you would use the actual URL from the API
      const fileDownloadUrl = responseData.downloadUrl || URL.createObjectURL(new Blob(['Processed data'], { type: 'text/csv' }));
      
      setDownloadUrl(fileDownloadUrl);
      setMessage('Processing complete! Your file is ready for download.');
    } catch (error) {
      setError(true);
      setMessage(`Error processing file: ${error.message || 'Unknown error'}`);
    } finally {
      setIsProcessing(false);
    }
  };

  const downloadFile = () => {
    if (downloadUrl) {
      // In a real app, you might want to use file-saver or a similar library
      const link = document.createElement('a');
      link.href = downloadUrl;
      link.download = `processed-${file.name}`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    }
  };

  return (
    <AppContainer>
      <Header>
        <Title>Product Details Dashboard</Title>
      </Header>
      
      <UploadSection>
        <HiddenFileInput 
          type="file" 
          id="file-upload" 
          accept=".csv,.xlsx,.xls" 
          onChange={handleFileChange} 
        />
        <FileLabel htmlFor="file-upload">
          Select Excel/CSV File
        </FileLabel>
        
        {file && (
          <FileName>
            {file.name}
          </FileName>
        )}
        
        {file && !isProcessing && !downloadUrl && (
          <div>
            <ProcessButton onClick={processFile} disabled={isProcessing}>
              Process File
            </ProcessButton>
          </div>
        )}
        
        {isProcessing && (
          <div>
            <ProgressContainer>
              <ProgressBar progress={progress} />
            </ProgressContainer>
            <StatusContainer>
              <StatusIcon processing />
              <StatusMessage>{message}</StatusMessage>
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
              Download Processed File
            </DownloadButton>
          </div>
        )}
        
        {error && (
          <StatusContainer>
            <StatusIcon error />
            <StatusMessage error>{message}</StatusMessage>
          </StatusContainer>
        )}
      </UploadSection>
    </AppContainer>
  );
}

export default App;