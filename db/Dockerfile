FROM mongo:4.4.29

ENV MONGO_INITDB_ROOT_USERNAME=root
ENV MONGO_INITDB_ROOT_PASSWORD=rootpassword
ENV MONGO_INITDB_DATABASE=aws-demo

# Copy the custom initialization scripts into the container
COPY init-mongo.js /docker-entrypoint-initdb.d/

EXPOSE 27017