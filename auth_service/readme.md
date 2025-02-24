## Auth Service

Authentication service shoulder responsibility for registration, authorization and storing personal users data.

The following handlers should be implemented:

- ```/api/v1/register``` - User sends their personal data in order to create an account,
namely ```login``` (Real company name if it is a business account), ```password```, ```phone_number```, ```first_name```,
```second_name```,  ```is_company```

- ```/api/v1/login``` - User sends their login and password. Auth service returns jwt token.

- ```/api/v1/get_user_info``` - get users' public data by ids list

- ```/api/v1/validate``` - accepts
