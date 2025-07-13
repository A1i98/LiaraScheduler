# Liara Scheduler

### Project Description
Liara Scheduler is a web application built with Go that allows users to automate the scaling (turning on/off) of their Liara.ir projects and databases based on predefined schedules using cron expressions. This tool is designed to help users manage their resource consumption efficiently by automatically scaling down services when not in use and scaling them back up when needed.

### Purpose and Goal
The primary goal of Liara Scheduler is to provide a convenient and automated way for Liara.ir users to optimize their costs and resource usage. By scheduling the on/off states of projects and databases, users can ensure that their services are only consuming resources when actively required, leading to potential cost savings and better resource management.

### Features
*   **Liara.ir API Integration:** Securely interacts with the Liara.ir API using user-provided tokens.
*   **Project and Database Management:** View a list of your Liara.ir projects and databases.
*   **Scheduled Scaling:** Define custom schedules using cron expressions to automatically turn projects and databases on or off.
*   **Web Interface:** A simple web-based user interface for easy interaction and schedule management.
*   **Schedule Monitoring:** View active schedules, their next run times, and last run times.
*   **Schedule Deletion:** Ability to remove existing schedules.
*   **Logging:** In-memory logging per user token to track actions and API responses.
*   **Uptime Monitoring:** Check the server's uptime.

### How It Works
The application runs as a web server. Users log in with their Liara API token, which is then used to authenticate all subsequent API calls to Liara.ir. Users can then create schedules by specifying a service (project or database), its name, an action (on/off), and a cron expression. The application uses a cron scheduler to execute these actions at the specified times, making API calls to Liara.ir to scale the services.

### Getting Started

#### Prerequisites
*   Go (version 1.16 or higher)
*   A Liara.ir account and API Token

#### Installation
1.  Clone the repository:
    ```bash
    git clone https://github.com/A1i98/LiaraScheduler.git
    cd LiaraScheduler
    ```
2.  Install Go dependencies:
    ```bash
    go mod tidy
    ```
3.  Create a `.env` file in the root directory and add your desired port (e.g., `PORT=8080`). If not specified, it defaults to `8080`.
    ```
    PORT=8080
    ```

#### Running the Application
```bash
go run main.go
```
The application will be accessible in your browser at `http://localhost:8080` (or your specified port).

### Contributing
Contributions are welcome! Please feel free to open issues or submit pull requests.

### License
This project is licensed under the MIT License. See the `LICENSE` file for details.
