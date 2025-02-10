## Stats service

Accepts statistic from kafka. It logs every comment, view and like event. Stats service expose 
REST API for stats reading.

The following handlers should be implemented:

- ```/api/v1/stats/post/{id}?(some query params for filtering)``` - post stats
- ```/api/v1/stats/comment/{id}?(some query params for filtering)``` - comment stats
- ```/api/v1/stats/user/{id}?(some query params for filtering)``` - comment stats
