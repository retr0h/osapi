---
sidebar_position: 5
---

# Guiding Principles

1. **Simplicity and Minimalism**

   Keep the codebase lightweight and minimalistic, focusing on core
   functionality without unnecessary complexity.

2. **Automation through OpenAPI**

   Use OpenAPI specifications to automate the generation of both APIs and
   clients.

3. **Pluggability and Extensibility**

   Design the system with pluggability in mind, allowing for easy extension and
   customization through well-defined interfaces and modules.

4. **Task Queue for Privileged Operations**

   Implement privileged system changes asynchronously through a task queue.

5. **RESTful API Design**

   Follow RESTful principles to design the API, supporting full CRUD (Create,
   Read, Update, Delete) operations for managing Linux system configurations.

6. **Reliability and Stability**

   Prioritize the stability and reliability of the API over features, ensuring
   it can safely manage critical system-level configurations.

7. **CLI Parity with API**

   Ensure that anything that can be accomplished through the API also has an
   equivalent option available via the CLI.
