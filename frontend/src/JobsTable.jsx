import React from 'react';
import styled from 'styled-components';

const TableContainer = styled.div`
  background-color: #f8f9fa;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-bottom: 2rem;
`;

const Table = styled.table`
  width: 100%;
  border-collapse: collapse;
  margin-top: 1rem;
`;

const TableHead = styled.thead`
  background-color: #f1f3f4;
`;

const TableRow = styled.tr`
  &:nth-child(even) {
    background-color: #f9f9f9;
  }
  
  &:hover {
    background-color: #f1f1f1;
  }
`;

const TableHeader = styled.th`
  padding: 0.75rem;
  text-align: left;
  border-bottom: 2px solid #e0e0e0;
`;

const TableCell = styled.td`
  padding: 0.75rem;
  border-bottom: 1px solid #e0e0e0;
`;

const StatusBadge = styled.span`
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: bold;
  text-transform: uppercase;
  
  ${props => props.status === 'completed' && `
    background-color: #e6f4ea;
    color: #0f9d58;
  `}
  
  ${props => props.status === 'processing' && `
    background-color: #e8f0fe;
    color: #4285f4;
  `}
  
  ${props => props.status === 'failed' && `
    background-color: #fce8e6;
    color: #ea4335;
  `}
  
  ${props => props.status === 'pending' && `
    background-color: #fef7e0;
    color: #f9ab00;
  `}
`;

const ProgressContainer = styled.div`
  width: 100%;
  height: 8px;
  background-color: #e0e0e0;
  border-radius: 4px;
  overflow: hidden;
`;

const ProgressBar = styled.div`
  height: 100%;
  background-color: #4285f4;
  width: ${props => props.progress || 0}%;
  transition: width 0.3s ease-in-out;
`;

const RefreshButton = styled.button`
  background-color: #4285f4;
  color: white;
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
  margin-bottom: 1rem;
  
  &:hover {
    background-color: #3367d6;
  }
`;

const DownloadButton = styled.a`
  display: inline-block;
  background-color: #0f9d58;
  color: white;
  padding: 0.25rem 0.75rem;
  border-radius: 4px;
  text-decoration: none;
  font-size: 0.9rem;
  
  &:hover {
    background-color: #0b8043;
  }
`;

const EmptyState = styled.div`
  text-align: center;
  padding: 2rem;
  color: #666;
`;

const JobsTable = ({ jobs, onRefresh }) => {
  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };
  
  return (
    <TableContainer>
      <RefreshButton onClick={onRefresh}>
        Refresh Jobs
      </RefreshButton>
      
      {Object.keys(jobs).length === 0 ? (
        <EmptyState>
          <p>No jobs found. Upload a file to start processing.</p>
        </EmptyState>
      ) : (
        <Table>
          <TableHead>
            <TableRow>
              <TableHeader>Job ID</TableHeader>
              <TableHeader>File Name</TableHeader>
              <TableHeader>Start Time</TableHeader>
              <TableHeader>Status</TableHeader>
              <TableHeader>Progress</TableHeader>
              <TableHeader>Actions</TableHeader>
            </TableRow>
          </TableHead>
          <tbody>
            {Object.entries(jobs).map(([jobId, jobData]) => (
              <TableRow key={jobId}>
                <TableCell>{jobId}</TableCell>
                <TableCell>{jobData.file_name}</TableCell>
                <TableCell>{formatDate(jobData.start_time)}</TableCell>
                <TableCell>
                  <StatusBadge status={jobData.status}>
                    {jobData.status}
                  </StatusBadge>
                </TableCell>
                <TableCell>
                  <ProgressContainer>
                    <ProgressBar progress={jobData.progress_percentage} />
                  </ProgressContainer>
                  <span>{jobData.progress_percentage}% ({jobData.processed} of {jobData.total})</span>
                </TableCell>
                <TableCell>
                  {jobData.status === 'completed' && (
                    <DownloadButton href={`http://localhost:8000/download/${jobId}`}>
                      Download
                    </DownloadButton>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </tbody>
        </Table>
      )}
    </TableContainer>
  );
};

export default JobsTable;
