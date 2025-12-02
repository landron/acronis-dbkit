-- init.mssql.sql
CREATE DATABASE dbkit_test;
GO

-- Create a login and user, and grant permissions
USE dbkit_test;
CREATE LOGIN admin WITH PASSWORD = 'qwe123QWE';
CREATE USER admin FOR LOGIN admin;
ALTER ROLE db_owner ADD MEMBER admin;
GO
