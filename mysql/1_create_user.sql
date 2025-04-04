CREATE USER 'coxwave'@'%' IDENTIFIED WITH mysql_native_password BY 'coxwavewave';

# main database
CREATE DATABASE coupons;
GRANT ALL PRIVILEGES ON coupons.* TO 'coxwave'@'%';

# unit-test database
CREATE DATABASE coupons_test;
GRANT ALL PRIVILEGES ON coupons_test.* TO 'coxwave'@'%';
