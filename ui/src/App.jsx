import React, { useState } from 'react';

export default function App() {
  const [file, setFile] = useState(null);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [response, setResponse] = useState(null);
  const [error, setError] = useState(null);
  
  const handleFileChange = (e) => {
    setFile(e.target.files[0]);
    setError(null);
    setResponse(null);
    setUploadProgress(0);
  };
  
  const handleSubmit = (e) => {
    e.preventDefault();
    if (!file) {
      setError("Please select a file first");
      return;
    }
    
    setUploading(true);
    setError(null);
    setUploadProgress(0);
    
    const formData = new FormData();
    formData.append('file', file);
    formData.append('openNow', 'true');
    
    // Create a new XMLHttpRequest to track upload progress
    const xhr = new XMLHttpRequest();
    
    // Track upload progress
    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable) {
        const progressPercent = Math.round((event.loaded / event.total) * 100);
        setUploadProgress(progressPercent);
      }
    });
    
    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const data = JSON.parse(xhr.responseText);
          setResponse(data);
          
          if (!data.success) {
            setError(data.errorMessage || "Upload failed");
            console.error("Upload error:", data.errorMessage);
          }
        } catch (err) {
          setError("Failed to parse response");
          console.error("Parse error:", err);
        }
      } else {
        setError(`Server error: ${xhr.status}`);
        console.error("Server error:", xhr.status);
      }
      setUploading(false);
    });
    
    xhr.addEventListener('error', () => {
      setError("Network error occurred");
      setResponse(null);
      console.error("Network error");
      setUploading(false);
    });
    
    xhr.addEventListener('abort', () => {
      setError("Upload aborted");
      setResponse(null);
      console.error("Upload aborted");
      setUploading(false);
    });
    
    // Open and send the request
    xhr.open('POST', 'http://localhost:8080/upload');
    xhr.send(formData);
  };
  
  return (
    <div className="flex flex-col items-center p-6 max-w-md mx-auto bg-white rounded-xl shadow-md">
      <h2 className="text-2xl font-bold mb-6 text-gray-800">File Opener Demo</h2>
      
      <div className="w-full">
        <div className="mb-4">
          <p className="block text-gray-700 text-sm font-bold mb-2">
            Select File to Upload and Open
          </p>
          
          <div className="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center">
            <input 
              type="file" 
              id="file"
              onChange={handleFileChange} 
              className="hidden" 
              disabled={uploading}
            />
            <label 
              htmlFor="file" 
              className={`cursor-pointer text-blue-500 hover:text-blue-700 ${uploading ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              {file ? file.name : "Click to browse files"}
            </label>
            
            {file && (
              <div className="mt-2 text-sm text-gray-600">
                {(file.size / 1024 / 1024).toFixed(2)} MB
              </div>
            )}
          </div>
        </div>
        
        {uploading && (
          <div className="mb-4">
            <div className="w-full bg-gray-200 rounded-full h-2.5">
              <div 
                className="bg-blue-600 h-2.5 rounded-full transition-all duration-300" 
                style={{ width: `${uploadProgress}%` }}
              ></div>
            </div>
            <div className="mt-1 text-center text-sm text-gray-600">
              {uploadProgress}% Uploaded
            </div>
          </div>
        )}
        
        <button 
          onClick={handleSubmit}
          disabled={!file || uploading}
          className={`w-full py-2 px-4 rounded-lg font-bold ${
            !file || uploading
              ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
              : 'bg-blue-500 hover:bg-blue-700 text-white'
          }`}
        >
          {uploading ? (
            <div className="flex items-center justify-center">
              <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-gray-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              Uploading...
            </div>
          ) : 'Upload and Open File'}
        </button>
      </div>
      
      {error && (
        <div className="mt-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
          {error}
        </div>
      )}
      
      {response && response.success && (
        <div className="mt-4 p-3 bg-green-100 border border-green-400 text-green-700 rounded">
          <p>File uploaded successfully!</p>
          <p className="text-xs mt-1">Path: {response.filePath}</p>
        </div>
      )}
      
      <div className="mt-6 text-sm text-gray-600">
        <p>This demo connects to your local Pop OS file opener agent.</p>
        <p>Make sure the agent is running on <code>http://localhost:8080</code></p>
      </div>
    </div>
  );
}
