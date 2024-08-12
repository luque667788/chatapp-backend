# Realtime Chat App Backend

This repository contains the backend code for a realtime chat application. The backend is written in Go and uses WebSockets for real-time communication and Redis for data storage. This project was developed as part of my learning journey in software development, and while it implements basic functionality, it may not follow all best practices.

## Features

- **User Handling and Authentication**: Basic user management and authentication mechanisms.
- **Message Saving**: Stores chat messages for retrieval.
- **WebSocket Communication**: Real-time messaging using WebSockets.
- **Redis Integration**: Uses Redis for data storage and caching.
- **Docker Deployment**: Easily deployable using Docker.


## Key Files and Directories

- **`main.go`**: Entry point of the application. Sets up routes and starts the server.
- **`pkg/websocket/`**: Contains the WebSocket implementation.
  - **`client.go`**: Manages individual WebSocket connections.
  - **`helpers.go`**: Utility functions for WebSocket operations.
  - **`pool.go`**: Manages a pool of WebSocket connections.
  - **`structs.go`**: Defines data structures used in WebSocket communication.
  - **`websocket.go`**: Initializes WebSocket connections and handles incoming requests.
- **`Dockerfile`**: Docker configuration for building and running the application.
- **`go.mod`**: Go module file listing dependencies.

## Getting Started

### Prerequisites

- Go 1.17 or later
- Docker

### Running the Application

1. Clone the repository:

   ```bash
   git clone https://github.com/luque667788/chatapp-backend.git
   cd chatapp-backend
   ```

2. Build and run using Docker:

   ```bash
   docker build -t chatapp-backend .
   docker run -p 8080:8080 chatapp-backend
   ```

3. Access the application: Open your browser and navigate to `http://localhost:8080`.

Remember to also clone and run the [frontend repo](https://github.com/luque667788/webdev).

## Notes

- This project was developed early in my learning curve as a software developer. As such, it may not follow all best practices.
- There is a frontend repository available that complements this backend. You can find it [here](https://github.com/luque667788/webdev).

## Contributing

Feel free to fork this repository and submit pull requests. Any contributions to improve the code quality and add new features are welcome!

## License

This project is licensed under the MIT License.

Thank you for checking out this project! If you have any questions or feedback, feel free to open an issue or contact me directly.
https://www.linkedin.com/in/luiz-henrique-salles-de-oliveira-mendon%C3%A7a-3963b928b/
