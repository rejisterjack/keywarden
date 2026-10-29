CREATE USER orders_owner WITH PASSWORD 'fake-orders-password';
CREATE DATABASE orders OWNER orders_owner;
REVOKE ALL ON DATABASE keywarden FROM PUBLIC;
GRANT CONNECT, TEMP ON DATABASE keywarden TO keywarden;
