import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import useWebSocket from 'react-use-websocket';

const API_BASE_URL = 'http://localhost:8080';
const WS_URL = 'ws://localhost:8080/ws';

function App() {
  const [cacheKey, setCacheKey] = useState('');
  const [cacheValue, setCacheValue] = useState('');
  const [expiration, setExpiration] = useState(60);
  const [apiResult, setApiResult] = useState(null);
  const [error, setError] = useState(null);
  const [cacheItems, setCacheItems] = useState({});

  const { lastMessage } = useWebSocket(WS_URL, {
    onOpen: () => console.log('WebSocket Connected'),
    onError: (event) => console.error('WebSocket Error', event),
    shouldReconnect: (closeEvent) => true,
  });

  useEffect(() => {
    if (lastMessage !== null) {
      const cacheUpdate = JSON.parse(lastMessage.data);
      console.log('WebSocket message received:', cacheUpdate);
      setCacheItems((prevItems) => {
        if (cacheUpdate.value === null) {
          const { [cacheUpdate.key]: _, ...rest } = prevItems;
          console.log('Deleting key from cache:', cacheUpdate.key);
          return rest;
        } else {
          console.log('Updating/Adding key to cache:', cacheUpdate.key, cacheUpdate.value);
          return {
            ...prevItems,
            [cacheUpdate.key]: {
              value: cacheUpdate.value,
              expiresAt: new Date(cacheUpdate.expiresAt),
            },
          };
        }
      });
    }
  }, [lastMessage]);

  const syncCache = useCallback(async () => {
    try {
      const response = await axios.get(`${API_BASE_URL}/cache`);
      setCacheItems(response.data);
    } catch (err) {
      console.error('Failed to sync cache:', err);
      setError(`Failed to sync cache: ${err.message}`);
      if (err.response) {
        console.error('Response data:', err.response.data);
        console.error('Response status:', err.response.status);
        console.error('Response headers:', err.response.headers);
      }
    }
  }, []);

  useEffect(() => {
    syncCache();
    const intervalId = setInterval(syncCache, 30000);
    return () => clearInterval(intervalId);
  }, [syncCache]);

  const handleGet = useCallback(async () => {
    try {
      setError(null);
      const response = await axios.get(`${API_BASE_URL}/cache/${cacheKey}`);
      setApiResult(response.data);
    } catch (err) {
      setError(err.response?.data || 'An error occurred while fetching the data');
      setApiResult(null);
    }
  }, [cacheKey]);

  const handleSet = useCallback(async () => {
    try {
      setError(null);
      const response = await axios.post(`${API_BASE_URL}/cache`, {
        key: cacheKey,
        value: cacheValue,
        expiration: parseInt(expiration),
      });
      setApiResult(response.data);
    } catch (err) {
      setError(err.response?.data || 'An error occurred while setting the key');
      setApiResult(null);
    }
  }, [cacheKey, cacheValue, expiration]);

  const handleDelete = useCallback(async () => {
    try {
      setError(null);
      const response = await axios.delete(`${API_BASE_URL}/cache/${cacheKey}`);
      setApiResult(response.data);
    } catch (err) {
      setError(err.response?.data || 'An error occurred while deleting the key');
      setApiResult(null);
    }
  }, [cacheKey]);

  return (
    <div className="min-h-screen bg-gray-100 py-6 flex flex-col justify-center sm:py-12">
      <div className="relative py-3 sm:max-w-xl sm:mx-auto">
        <div className="absolute inset-0 bg-gradient-to-r from-cyan-400 to-light-blue-500 shadow-lg transform -skew-y-6 sm:skew-y-0 sm:-rotate-6 sm:rounded-3xl"></div>
        <div className="relative px-4 py-10 bg-white shadow-lg sm:rounded-3xl sm:p-20">
          <h1 className="text-4xl font-bold mb-8 text-center text-gray-800">LRU Cache Interface</h1>
          
          <div className="mb-8 space-y-4">
            <input
              type="text"
              placeholder="Key"
              value={cacheKey}
              onChange={(e) => setCacheKey(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-cyan-500"
            />
            <input
              type="text"
              placeholder="Value"
              value={cacheValue}
              onChange={(e) => setCacheValue(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-cyan-500"
            />
            <input
              type="number"
              placeholder="Expiration (seconds)"
              value={expiration}
              onChange={(e) => setExpiration(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-cyan-500"
            />
            <div className="flex space-x-2">
              <button onClick={handleGet} className="flex-1 bg-cyan-500 hover:bg-cyan-600 text-white font-bold py-2 px-4 rounded-md transition duration-300">Get</button>
              <button onClick={handleSet} className="flex-1 bg-green-500 hover:bg-green-600 text-white font-bold py-2 px-4 rounded-md transition duration-300">Set</button>
              <button onClick={handleDelete} className="flex-1 bg-red-500 hover:bg-red-600 text-white font-bold py-2 px-4 rounded-md transition duration-300">Delete</button>
            </div>
            <button onClick={syncCache} className="w-full bg-yellow-500 hover:bg-yellow-600 text-white font-bold py-2 px-4 rounded-md transition duration-300">Sync Cache</button>
          </div>

          {error && (
            <div className="bg-red-100 border-l-4 border-red-500 text-red-700 p-4 mb-8" role="alert">
              <p className="font-bold">Error</p>
              <p>{error}</p>
            </div>
          )}

          {apiResult && (
            <div className="mb-8">
              <h2 className="text-2xl font-semibold mb-2 text-gray-700">API Result:</h2>
              <pre className="bg-gray-100 p-4 rounded-md overflow-x-auto">{JSON.stringify(apiResult, null, 2)}</pre>
            </div>
          )}

          <div>
            <h2 className="text-2xl font-semibold mb-4 text-gray-700">Current Cache Contents:</h2>
            {Object.entries(cacheItems).length === 0 ? (
              <p className="text-gray-600">Cache is empty</p>
            ) : (
              <ul className="space-y-4">
                {Object.entries(cacheItems).map(([key, { value, expiresAt }]) => (
                  <li key={key} className="bg-gray-50 p-4 rounded-md shadow">
                    <div className="font-semibold text-gray-800">{key}:</div>
                    <div className="text-gray-600 mt-1">{JSON.stringify(value)}</div>
                    <div className="text-sm text-gray-500 mt-2">
                      Expires: {new Date(expiresAt).toLocaleString()}
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </div>
    </div>
  );
  
}

export default App;