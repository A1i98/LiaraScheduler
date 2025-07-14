document.addEventListener('DOMContentLoaded', () => {
    const loginSection = document.getElementById('login-section');
    const mainAppSection = document.getElementById('main-app-section');
    const loginForm = document.getElementById('login-form');
    const tokenInput = document.getElementById('token-input');
    const loginError = document.getElementById('login-error');

    const tabButtons = document.querySelectorAll('.tab-button');
    const tabContents = document.querySelectorAll('.tab-content');

    // Project elements
    const projectStatusList = document.getElementById('project-status-list');
    const projectSelect = document.getElementById('project-select');
    const projectError = document.getElementById('project-error');
    const scheduleProjectForm = document.getElementById('schedule-project-form');

    // Database elements
    const databaseStatusList = document.getElementById('database-status-list');
    const databaseSelect = document.getElementById('database-select');
    const databaseError = document.getElementById('database-error');
    const scheduleDatabaseForm = document.getElementById('schedule-database-form');

    // Schedule elements
    const currentTimeDisplay = document.getElementById('current-time-display');
    const currentSchedulesList = document.getElementById('current-schedules');

    // Log and Uptime elements
    const serverLogsPre = document.getElementById('server-logs');
    const serverUptimeP = document.getElementById('server-uptime');

    let liaraToken = localStorage.getItem('liaraToken');

    if (liaraToken) {
        showMainAppSection();
        loadAllData();
    } else {
        loginSection.style.display = 'block';
        mainAppSection.style.display = 'none';
    }

    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const token = tokenInput.value;
        const response = await fetch('/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ token: token }),
        });

        if (response.ok) {
            localStorage.setItem('liaraToken', token);
            liaraToken = token;
            loginError.textContent = '';
            showMainAppSection();
            loadAllData();
        } else {
            const errorData = await response.json();
            loginError.textContent = errorData.error || 'Login failed.';
        }
    });

    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            const tab = button.dataset.tab;

            tabContents.forEach(content => {
                content.classList.remove('active');
            });
            tabButtons.forEach(btn => {
                btn.classList.remove('active');
            });

            document.getElementById(`${tab}-tab`).classList.add('active');
            button.classList.add('active');

            // Fetch data specific to the tab when it's activated
            if (tab === 'projects') {
                fetchProjects();
            } else if (tab === 'databases') {
                fetchDatabases();
            } else if (tab === 'schedules') {
                fetchSchedules();
            } else if (tab === 'logs') {
                fetchLogs();
            } else if (tab === 'uptime') {
                fetchUptime();
            }
        });
    });

    scheduleProjectForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const selectedProject = projectSelect.value;
        const action = document.getElementById('project-action-select').value;
        const cron = document.getElementById('project-cron-input').value;

        if (!selectedProject) {
            projectError.textContent = 'Please select a project.';
            return;
        }
        projectError.textContent = '';

        const response = await fetch('/schedule', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${liaraToken}`
            },
            body: JSON.stringify({ service: selectedProject, serviceType: "project", action, cron }),
        });

        if (response.ok) {
            alert('Project schedule added successfully!');
            document.getElementById('project-cron-input').value = '';
            fetchSchedules();
        } else {
            const errorData = await response.json();
            alert(`Failed to add project schedule: ${errorData.error || 'Unknown error'}`);
        }
    });

    scheduleDatabaseForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const selectedDatabase = databaseSelect.value;
        const action = document.getElementById('database-action-select').value;
        const cron = document.getElementById('database-cron-input').value;

        if (!selectedDatabase) {
            databaseError.textContent = 'Please select a database.';
            return;
        }
        databaseError.textContent = '';

        const response = await fetch('/schedule', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${liaraToken}`
            },
            body: JSON.stringify({ service: selectedDatabase, serviceType: "database", action, cron }),
        });

        if (response.ok) {
            alert('Database schedule added successfully!');
            document.getElementById('database-cron-input').value = '';
            fetchSchedules();
        } else {
            const errorData = await response.json();
            alert(`Failed to add database schedule: ${errorData.error || 'Unknown error'}`);
        }
    });

    function showMainAppSection() {
        loginSection.style.display = 'none';
        mainAppSection.style.display = 'block';
    }

    async function loadAllData() {
        fetchProjects();
        fetchDatabases();
        fetchSchedules();
        fetchLogs();
        fetchUptime();
    }

    async function fetchProjects() {
        projectStatusList.innerHTML = '<li>Loading projects...</li>';
        projectSelect.innerHTML = '<option value="">Loading projects...</option>';
        try {
            const response = await fetch('/projects', {
                headers: {
                    'Authorization': `Bearer ${liaraToken}`
                }
            });
            if (response.ok) {
                const projects = await response.json();
                projectStatusList.innerHTML = '';
                projectSelect.innerHTML = '';
                if (projects.length === 0) {
                    projectStatusList.innerHTML = '<li>No projects found.</li>';
                    projectSelect.innerHTML = '<option value="">No projects found</option>';
                } else {
                    projects.forEach(project => {
                        const li = document.createElement('li');
                        li.textContent = `ID: ${project.project_id} | Type: ${project.type} | Status: ${project.status} | Scale: ${project.scale}`;
                        projectStatusList.appendChild(li);

                        const option = document.createElement('option');
                        option.value = project.project_id;
                        option.textContent = project.project_id;
                        projectSelect.appendChild(option);
                    });
                }
            } else {
                const errorData = await response.json();
                projectStatusList.innerHTML = '<li>Error loading projects.</li>';
                projectSelect.innerHTML = '<option value="">Error loading projects</option>';
                projectError.textContent = errorData.error || 'Failed to fetch projects.';
                console.error('Failed to fetch projects:', errorData.error);
            }
        } catch (error) {
            projectStatusList.innerHTML = '<li>Network error or server unavailable.</li>';
            projectSelect.innerHTML = '<option value="">Error loading projects</option>';
            projectError.textContent = 'Network error or server unavailable.';
            console.error('Network error:', error);
        }
    }

    async function fetchDatabases() {
        databaseStatusList.innerHTML = '<li>Loading databases...</li>';
        databaseSelect.innerHTML = '<option value="">Loading databases...</option>';
        try {
            const response = await fetch('/databases', {
                headers: {
                    'Authorization': `Bearer ${liaraToken}`
                }
            });
            if (response.ok) {
                const databases = await response.json();
                databaseStatusList.innerHTML = '';
                databaseSelect.innerHTML = '<option value="">Select a database...</option>';
                if (databases.length === 0) {
                    databaseStatusList.innerHTML = '<li>No databases found.</li>';
                    databaseSelect.innerHTML = '<option value="">No databases found</option>';
                } else {
                    databases.forEach(db => {
                        const li = document.createElement('li');
                        li.textContent = `ID: ${db.DBId} | Type: ${db.type} | Status: ${db.status} | Scale: ${db.scale} | Host: ${db.hostname}`;
                        databaseStatusList.appendChild(li);

                        const option = document.createElement('option');
                        option.value = db.DBId || '';
                        option.textContent = `${db.DBId || ''} (${db.type} - ${db.hostname})`;
                        databaseSelect.appendChild(option);
                    });
                }
            } else {
                const errorData = await response.json();
                databaseStatusList.innerHTML = '<li>Error loading databases.</li>';
                databaseSelect.innerHTML = '<option value="">Error loading databases</option>';
                databaseError.textContent = errorData.error || 'Failed to fetch databases.';
                console.error('Failed to fetch databases:', errorData.error);
            }
        } catch (error) {
            databaseStatusList.innerHTML = '<li>Network error or server unavailable.</li>';
            databaseSelect.innerHTML = '<option value="">Error loading databases</option>';
            databaseError.textContent = 'Network error or server unavailable.';
            console.error('Network error:', error);
        }
    }

    async function fetchSchedules() {
        currentSchedulesList.innerHTML = '<li>Loading schedules...</li>';
        try {
            const response = await fetch('/schedules', {
                headers: {
                    'Authorization': `Bearer ${liaraToken}`
                }
            });
            if (response.ok) {
                const data = await response.json();
                const schedules = data.schedules;
                const currentTime = new Date(data.currentTime);

                currentTimeDisplay.textContent = `Current Time: ${formatDate(currentTime)}`;

                currentSchedulesList.innerHTML = '';
                if (schedules.length === 0) {
                    currentSchedulesList.innerHTML = '<li>No schedules added yet.</li>';
                } else {
                    schedules.forEach(schedule => {
                        const li = document.createElement('li');
                        let scheduleText = `Service: ${schedule.ServiceName} (${schedule.ServiceType}) | Action: ${schedule.Action} | Cron: ${schedule.CronSpec}`;

                        if (schedule.LastRun) {
                            scheduleText += ` | Last Run: ${formatDate(new Date(schedule.LastRun))}`;
                        }
                        if (schedule.NextRun) {
                            scheduleText += ` | Next Run: ${formatDate(new Date(schedule.NextRun))}`;
                        }

                        li.textContent = scheduleText;

                        const deleteButton = document.createElement('button');
                        deleteButton.textContent = 'Delete';
                        deleteButton.classList.add('delete-button');
                        deleteButton.addEventListener('click', async () => {
                            if (confirm(`Are you sure you want to delete the schedule for ${schedule.ServiceType} "${schedule.ServiceName}"?`)) {
                                await deleteSchedule(schedule.JobID);
                            }
                        });
                        li.appendChild(deleteButton);
                        currentSchedulesList.appendChild(li);
                    });
                }
            } else {
                const errorData = await response.json();
                currentSchedulesList.innerHTML = '<li>Error loading schedules.</li>';
                console.error('Failed to fetch schedules:', errorData.error);
            }
        } catch (error) {
            currentSchedulesList.innerHTML = '<li>Network error or server unavailable.</li>';
            console.error('Network error:', error);
        }
    }

    async function deleteSchedule(jobID) {
        try {
            const response = await fetch(`/schedule/delete/${jobID}`, {
                method: 'DELETE',
                headers: {
                    'Authorization': `Bearer ${liaraToken}`
                }
            });

            if (response.ok) {
                alert('Schedule deleted successfully!');
                fetchSchedules();
            } else {
                const errorData = await response.json();
                alert(`Failed to delete schedule: ${errorData.error || 'Unknown error'}`);
                console.error('Failed to delete schedule:', errorData.error);
            }
        } catch (error) {
            alert('Network error or server unavailable.');
            console.error('Network error:', error);
        }
    }

    async function fetchLogs() {
        serverLogsPre.textContent = 'Loading logs...';
        try {
            const response = await fetch('/logs', {
                headers: {
                    'Authorization': `Bearer ${liaraToken}`
                }
            });
            if (response.ok) {
                const logs = await response.text();
                serverLogsPre.textContent = logs;
            } else {
                const errorData = await response.text();
                serverLogsPre.textContent = `Error loading logs: ${errorData || 'Unknown error'}`;
                console.error('Failed to fetch logs:', errorData);
            }
        } catch (error) {
            serverLogsPre.textContent = 'Network error or server unavailable.';
            console.error('Network error:', error);
        }
    }

    async function fetchUptime() {
        serverUptimeP.textContent = 'Loading uptime...';
        try {
            const response = await fetch('/uptime', {
                headers: {
                    'Authorization': `Bearer ${liaraToken}`
                }
            });
            if (response.ok) {
                const data = await response.json();
                serverUptimeP.textContent = `Server has been running for: ${data.uptime}`;
            } else {
                const errorData = await response.json();
                serverUptimeP.textContent = `Error loading uptime: ${errorData.error || 'Unknown error'}`;
                console.error('Failed to fetch uptime:', errorData.error);
            }
        } catch (error) {
            serverUptimeP.textContent = 'Network error or server unavailable.';
            console.error('Network error:', error);
        }
    }

    function formatDate(date) {
        const options = {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            hour12: false
        };
        return date.toLocaleString('en-US', options);
    }

    // Refresh data periodically
    setInterval(loadAllData, 10000);
});
