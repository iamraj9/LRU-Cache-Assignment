# LRU Cache API and Client

This repository contains two main folders:

1. **lru-cache-api**: GoLang backend
2. **lru-cache-client**: React JS frontend

## lru-cache-api (GoLang Backend)

### Overview

This folder contains the GoLang backend for the LRU Cache API.

### Getting Started

1. **Navigate to the `lru-cache-api` folder**:
    ```bash
    cd lru-cache-api
    ```

2. **Install dependencies** (if applicable):
    ```bash
    go mod tidy
    ```

3. **Build and run the GoLang application**:
    ```bash
    go run main.go
    ```

    Ensure that your GoLang environment is properly set up and the necessary environment variables are configured.

4. **Access the API**:
    - The API should be running on `http://localhost:8080` (or the port specified in your `main.go` file).

## lru-cache-client (React JS Frontend)

### Overview

This folder contains the React JS frontend for the LRU Cache Client.

### Getting Started

1. **Navigate to the `lru-cache-client` folder**:
    ```bash
    cd lru-cache-client
    ```

2. **Install dependencies**:
    ```bash
    npm install
    ```

3. **Run the React JS application**:
    ```bash
    npm start
    ```

    This will start the React application and open it in your default web browser.

4. **Access the client**:
    - The frontend should be running on `http://localhost:3000`.

## Running Both Projects

To ensure both the backend and frontend work correctly, you need to run them in parallel:

1. Start the GoLang backend:
    ```bash
    cd lru-cache-api
    go run main.go
    ```

2. In a separate terminal, start the React JS frontend:
    ```bash
    cd lru-cache-client
    npm start
    ```

## Contributing

Feel free to open issues or pull requests if you have suggestions or improvements. Please make sure to follow the contribution guidelines.

