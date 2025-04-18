const API_BASE_URL = 'http://localhost:8000'; // Update this if your backend runs on a different port

export const API_ENDPOINTS = {
  UPLOAD: `${API_BASE_URL}/upload/`,
  STATUS: (jobId) => `${API_BASE_URL}/status/${jobId}`,
  DOWNLOAD: (jobId) => `${API_BASE_URL}/download/${jobId}`,
  JOBS: `${API_BASE_URL}/jobs`
};

export const axiosConfig = {
  headers: {
    'Content-Type': 'multipart/form-data',
  },
}; 