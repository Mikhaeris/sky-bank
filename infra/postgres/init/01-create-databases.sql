CREATE USER auth_user WITH PASSWORD 'auth_password';
CREATE USER customer_user WITH PASSWORD 'customer_password';

CREATE DATABASE auth_db OWNER auth_user;
CREATE DATABASE customer_db OWNER customer_user;
