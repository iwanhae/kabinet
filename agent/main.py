import os
import requests
import json
from openai import OpenAI

# --- Configuration ---
# Kube Event Analyzer API Endpoint URL (from environment variable or default)
API_URL = os.getenv("KUBE_EVENT_ANALYZER_URL", "http://127.0.0.1:8080/query")
# Initialize OpenAI API Client (from environment variables)
# export OPENAI_API_KEY="your_api_key"
# export OPENAI_API_BASE="your_api_base_url" (if needed)
try:
    client = OpenAI(
        api_key=os.getenv("OPENAI_API_KEY"),
        base_url=os.getenv("OPENAI_API_BASE")
    )
    if not os.getenv("OPENAI_API_KEY"):
        raise ValueError("OPENAI_API_KEY environment variable is not set.")
except Exception as e:
    print(f"Failed to initialize OpenAI API client: {e}")
    exit()

# --- System Prompt ---
SYSTEM_PROMPT = """
You are a proactive and autonomous expert AI assistant for troubleshooting Kubernetes cluster events.
Your primary goal is to independently investigate user issues from start to finish by executing a series of SQL queries and analyzing the results. You lead the investigation.

**Your Core Mission**
A user will state a problem. You will then take charge of the entire investigation.
1.  **Form a Hypothesis**: Based on the user's request and the data you've gathered, form a hypothesis about the root cause.
2.  **Generate Queries to Test Hypothesis**: Generate a sequence of SQL queries to prove or disprove your hypothesis. Do not ask for permission.
3.  **Analyze and Iterate**: After I provide the JSON result for your query, analyze it. If your hypothesis was wrong or you need more data, form a new hypothesis and generate the next query.
4.  **Conclude with Actionable Insights**: When your investigation is complete, provide a final, comprehensive analysis in English. Explain what you found, what it means, and what the user should do next.

**Crucial Investigation Principles**
-   **Chronology is Key**: Don't just count events. Always analyze them in chronological order (`ORDER BY lastTimestamp`) to understand the sequence of cause and effect. This is the most important principle.
-   **Drill Down**: Start with a broad query. Based on the result, narrow your focus. For example, if you find a problematic node, your next query should be about *that specific node's events over time*.
-   **Be Proactive, Not Passive**: Never ask the user what to do next. You are the expert. Drive the investigation.

**Advanced Analysis Technique: Inferring Root Cause from Correlation**
You cannot directly measure disk usage of a pod from event data. However, you can make strong, evidence-based inferences.
-   **Scenario**: If you detect a `FreeDiskSpaceFailed` or `NodeHasDiskPressure` event on a node.
-   **Your Next Step**: Your hypothesis should be "The disk pressure was likely caused by large container images being pulled to the node."
-   **Test Query**: Immediately query for `Pulled` events on that *specific node* around the *specific time* of the failure. Look for image names that imply large size (e.g., containing 'cuda', 'jupyter', 'tensorflow', 'dind', 'ml-model').
    - `SELECT message, COUNT(*) as pull_count FROM $events WHERE source.host = '<node_name>' AND reason = 'Pulled' AND lastTimestamp BETWEEN '<start_time>' AND '<end_time>' GROUP BY message ORDER BY pull_count DESC`
-   **Conclusion**: If you find a correlation, present it as strong evidence. For example: "While I cannot directly measure disk usage, event analysis shows that numerous large CUDA images were pulled to this node right before the disk pressure event. This strongly suggests that these images are the root cause."

**Example Investigation Flow**
1.  *User*: "I think the cluster is unstable."
2.  *You*: ```sql SELECT reason, COUNT(*) FROM $events WHERE type = 'Warning' GROUP BY reason ORDER BY count DESC LIMIT 5; ```
3.  *Me*: (JSON result showing 'FreeDiskSpaceFailed' is a top reason)
4.  *You*: ```sql SELECT source.host, message, lastTimestamp FROM $events WHERE reason = 'FreeDiskSpaceFailed' ORDER BY lastTimestamp DESC; ```
5.  *Me*: (JSON result showing node 'node-xyz-123' had the issue at 05:51)
6.  *You*: ```sql SELECT reason, message, lastTimestamp FROM $events WHERE source.host = 'node-xyz-123' AND lastTimestamp BETWEEN '2025-08-03T05:40:00Z' AND '2025-08-03T06:00:00Z' ORDER BY lastTimestamp ASC; ```
7.  *Me*: (JSON result showing the timeline: NodeHasDiskPressure -> EvictionThresholdMet -> FreeDiskSpaceFailed)
8.  *You*: (Hypothesis: Caused by large images) ```sql SELECT message, COUNT(*) as pull_count FROM $events WHERE source.host = 'node-xyz-123' AND reason = 'Pulled' AND lastTimestamp BETWEEN '2025-08-03T05:00:00Z' AND '2025-08-03T06:00:00Z' GROUP BY message ORDER BY pull_count DESC LIMIT 10;```
9.  *Me*: (JSON result showing many `Pulled` events for images like `internal-registry.com/ml-project/cuda-base:latest`, `internal-registry.com/data-science/jupyter-notebook:v2.1`)
10. *You*: (Final analysis in English) "A disk space issue was detected on node 'node-xyz-123'. Disk pressure began at 05:48. Event log analysis revealed that **multiple large ML-related images were downloaded just before this time.** This strongly suggests that the large images used by specific ML workloads are the likely root cause of the disk space shortage. It is recommended to check the image sizes on the node using `docker images` or `crictl images` and clean up any unnecessary images."

**Event Schema Reference (`$events` table)**
- `metadata`: (name, namespace, creationTimestamp)
- `involvedObject`: (kind, namespace, name)
- `source`: (component, host)
- `reason`, `message`, `lastTimestamp`, `type`, `count`

You must now begin the investigation based on the user's request.
"""
def execute_query(query: str) -> dict:
    """Executes an SQL query by calling the Kube Event Analyzer API."""
    print(f"\n[INFO] Executing query:\n{query}")
    if not API_URL or "your-kube-event-analyzer-api.com" in API_URL:
        return {"error": "KUBE_EVENT_ANALYZER_URL environment variable is not set."}
    try:
        payload = {
            "start": "2000-01-01T00:00:00Z",
            "end": "2099-01-02T00:00:00Z",
            "query": query
        }
        response = requests.post(API_URL, json=payload, timeout=30)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        return {"error": f"API call failed: {e}"}
    except json.JSONDecodeError:
        return {"error": "API response is not valid JSON.", "content": response.text}

def get_ai_response(messages: list) -> str:
    """Fetches the AI's response from the OpenAI API."""
    print("\n[INFO] AI is analyzing and planning the next step...")
    completion = client.chat.completions.create(
        model="gpt-4.1",
        messages=messages
    )
    return completion.choices[0].message.content

def main():
    """Main function to handle the conversation with the AI agent."""
    print("🤖 Hello! I am an AI agent for analyzing Kubernetes cluster events.")
    print("Please ask me about the cluster status. (Type 'exit' to quit)")

    # A single list of messages to maintain the context of the entire conversation
    messages = [{"role": "system", "content": SYSTEM_PROMPT}]

    while True:
        user_input = input("\n[User] ")
        if user_input.lower() == 'exit':
            print("🤖 Exiting conversation. Thank you for using the service.")
            break
        
        # Add user input to the conversation history to start the investigation
        messages.append({"role": "user", "content": user_input})

        MAX_TURNS = 7 # Increased the maximum number of investigation turns
        for i in range(MAX_TURNS):
            print(f"\n===== [Investigation Step {i + 1}/{MAX_TURNS}] =====")
            
            ai_response_content = get_ai_response(messages)
            
            # Add the AI's response to the conversation history
            messages.append({"role": "assistant", "content": ai_response_content})

            # Check if the AI returned an SQL query for further investigation
            if "```sql" in ai_response_content:
                sql_query = ai_response_content.split("```sql")[1].split("```")[0].strip()
                
                query_result = execute_query(sql_query)
                if "error" in query_result:
                    print(f"\n[ERROR] An error occurred during query execution: {query_result['error']}")
                    messages.append({"role": "system", "content": f"Query execution failed: {query_result['error']}"})
                    continue

                result_str = json.dumps(query_result, indent=2, ensure_ascii=False)
                
                # If the result is too long, print a summary
                summary = result_str if len(result_str) < 1000 else result_str[:1000] + "..."
                print(f"\n[INFO] API Result Summary:\n{summary}")

                # Add the API result as a system message to the conversation history for the next analysis
                messages.append({
                    "role": "system",
                    "content": f"Query executed. Here is the JSON result. Analyze it, form a new hypothesis if needed, and decide whether to generate another query for deeper investigation or to provide a final analysis in English.\n{result_str}"
                })
            else:
                # If the AI returned a final analysis instead of an SQL query
                print(f"\n[AI Agent Final Analysis]\n{ai_response_content}")
                break # End the current investigation and wait for the next user input
        else:
            # If the maximum number of investigation steps has been reached
            print("\n🤖 Reached the maximum number of investigation steps. Providing analysis based on the information gathered so far.")
            # Request a summary based on the conversation so far
            messages.append({"role": "system", "content": "You have reached the maximum number of turns. Please summarize your findings so far in English and provide a final analysis."})
            final_summary = get_ai_response(messages)
            print(f"\n[AI Agent Final Summary]\n{final_summary}")


if __name__ == "__main__":
    main()

