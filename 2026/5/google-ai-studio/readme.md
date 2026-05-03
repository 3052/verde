# google ai studio

Since the billing and API are both active, the current API key is likely
restricted or corrupted from the billing migration. The cleanest fix is to
generate a new key directly inside this confirmed project.

Here are the exact clicks:

1. Go back to the **[Google AI Studio API Keys page](https://aistudio.google.com/app/apikey)**.
2. Click the **Create API key** button.
3. Click **Create API key in existing project**.
4. In the dropdown, select the exact project we were just looking at in the Cloud Console.
5. Click **Create API key**.

Copy that new key, replace the old one in your setup, and run your prompt. Let
me know if the internal error clears.
