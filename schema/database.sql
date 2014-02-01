CREATE DATABASE deckbrew WITH template=template0 encoding='UTF8'; 
CREATE USER urza WITH PASSWORD 'power9';
GRANT ALL PRIVILEGES ON DATABASE deckbrew TO urza;
