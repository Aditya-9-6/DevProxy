# Autonomous Agent System Setup Tasks

To ensure the autonomous agent system functions properly, you need to configure the required GitHub Secrets, set up a GitHub App for higher API rate limits, and ensure correct repository permissions. Please complete the following tasks:

## 1. Configure Gemini API Keys

The agents use Google Gemini as the LLM backend. You need to provide API keys as GitHub Secrets.

1. Go to your repository settings on GitHub.
2. Navigate to **Secrets and variables > Actions**.
3. Create the following repository secrets:
   - `GEMINI_API_KEY` (Required): Your primary Gemini API key.
   - `GEMINI_SOLVER_KEY` (Optional, overrides default for solving issues).
   - `GEMINI_REVIEWER_KEY` (Optional, overrides default for reviewing PRs).
   - `GEMINI_ISSUE_KEY` (Optional, overrides default for issue approval/management).
   - `GEMINI_KEY_POOL` (Optional, a comma-separated list of fallback API keys for failover).

*You can generate free API keys at [Google AI Studio](https://aistudio.google.com/).*

## 2. Set Up a GitHub App for Authentication

The workflows try to use a GitHub App for higher rate limits than the standard `GITHUB_TOKEN`. This prevents the bot from hitting API limits quickly.

1. Create a GitHub App in your developer settings.
2. Ensure it has the necessary permissions (Read/Write access for Issues, Pull Requests, Contents, and Workflows/Actions).
3. Install the app on this repository.
4. Go to **Secrets and variables > Actions** in the repository.
5. Create the following secrets:
   - `APP_ID`: The numeric ID of your GitHub App.
   - `APP_PRIVATE_KEY`: The private key generated for your GitHub App (in PEM format).

*If these are missing, the workflows will fall back to using `GITHUB_TOKEN`, but rate limits might be an issue.*

## 3. Verify Repository Permissions

Ensure GitHub Actions have the necessary permissions to push code and create/approve pull requests.

1. Go to your repository settings on GitHub.
2. Navigate to **Actions > General**.
3. Under **Workflow permissions**, ensure **Read and write permissions** is selected.
4. Ensure **Allow GitHub Actions to create and approve pull requests** is checked.
5. Save your changes.

Once these steps are completed, the agents (CI Repair Bot, Issue Solver, Issue Approver, and Auto-Merger) should be fully functional.
