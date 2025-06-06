# Family Service GraphQL 🌐

![GitHub release](https://img.shields.io/github/release/Digitaldecimator/family_service_hexarch_graphql.svg)
![GitHub issues](https://img.shields.io/github/issues/Digitaldecimator/family_service_hexarch_graphql.svg)
![GitHub stars](https://img.shields.io/github/stars/Digitaldecimator/family_service_hexarch_graphql.svg)

Welcome to the **Family Service GraphQL** repository! This project serves as an open-source starter template for a robust GraphQL service designed to manage family data. It employs a hexagonal architecture, allowing for flexibility and scalability. The service supports both MongoDB and PostgreSQL databases, making it versatile for various use cases.

## Table of Contents

- [Introduction](#introduction)
- [Features](#features)
- [Architecture](#architecture)
- [Technologies Used](#technologies-used)
- [Getting Started](#getting-started)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)
- [Links](#links)

## Introduction

In today’s digital world, managing family data efficiently is crucial. This repository provides a foundational framework to build upon. It allows developers to create a GraphQL service that is not only powerful but also easy to extend and maintain. 

## Features

- **GraphQL API**: Easily query and manipulate family data.
- **Hexagonal Architecture**: Promotes separation of concerns, making the codebase easier to manage.
- **Database Support**: Choose between MongoDB and PostgreSQL for data storage.
- **Monitoring**: Integrate with Grafana and Prometheus for performance monitoring.
- **Extensible**: Add new features without affecting existing functionality.

## Architecture

The hexagonal architecture separates the application into different layers, making it easier to test and maintain. The core business logic remains isolated from external systems like databases and APIs. This structure enhances adaptability and promotes cleaner code.

### Diagram

![Hexagonal Architecture](https://miro.medium.com/max/700/1*5Wf5t8Vq1XJcB2gI7n8n0A.png)

## Technologies Used

- **Go (Golang)**: The main programming language for the service.
- **GraphQL**: For building the API.
- **MongoDB**: A NoSQL database option.
- **PostgreSQL**: A relational database option.
- **Grafana**: For monitoring and visualization.
- **Prometheus**: For metrics collection.

## Getting Started

To get started with the Family Service GraphQL project, follow these steps:

1. **Clone the Repository**

   ```bash
   git clone https://github.com/Digitaldecimator/family_service_hexarch_graphql.git
   cd family_service_hexarch_graphql
   ```

2. **Install Dependencies**

   Ensure you have Go installed on your machine. Install the required packages by running:

   ```bash
   go mod tidy
   ```

3. **Configuration**

   Set up your environment variables for database connections. Create a `.env` file in the root directory and specify your database configurations.

   ```env
   DATABASE_TYPE=mongodb # or postgresql
   DATABASE_URL=your_database_url
   ```

4. **Run the Service**

   Start the service with the following command:

   ```bash
   go run main.go
   ```

5. **Access the GraphQL Playground**

   Open your browser and navigate to `http://localhost:8080/graphql` to access the GraphQL playground.

## Usage

You can use the GraphQL API to perform various operations on family data. Here are some example queries:

### Query Example

```graphql
query {
  families {
    id
    name
    members {
      id
      name
      age
    }
  }
}
```

### Mutation Example

```graphql
mutation {
  addFamily(name: "Smith") {
    id
    name
  }
}
```

## Contributing

We welcome contributions to improve the Family Service GraphQL project. To contribute, please follow these steps:

1. Fork the repository.
2. Create a new branch for your feature or bug fix.
3. Make your changes and commit them.
4. Push your branch to your fork.
5. Open a pull request.

Please ensure your code adheres to the existing style and includes tests where applicable.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Links

For the latest releases, please visit the [Releases](https://github.com/Digitaldecimator/family_service_hexarch_graphql/releases) section. Here, you can download and execute the latest version of the project.

For further updates, check back frequently or watch the repository to stay informed about new features and improvements.

---

Thank you for exploring the Family Service GraphQL project. We hope it serves as a valuable resource for your development needs!