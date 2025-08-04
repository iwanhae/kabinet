import os
import requests
import json
from openai import OpenAI
from datetime import datetime, timedelta, timezone

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
2.  **Generate Queries to Test Hypothesis**: Generate a sequence of queries to prove or disprove your hypothesis.
3.  **Analyze and Iterate**: After I provide the JSON result for your query, analyze it. If your hypothesis was wrong or you need more data, form a new hypothesis and generate the next query.
4.  **Conclude with Actionable Insights**: When your investigation is complete, provide a final, comprehensive analysis in English. Explain what you found, what it means, and what the user should do next.

**How to Generate Queries**
- **Query Format**: You MUST respond with a JSON object inside a `json` code block. Do not provide any other text or explanation outside of this block.
- **JSON Structure**:
  ```json
  {
    "query": "YOUR_SQL_QUERY",
    "start": "START_TIME_ISO_8601",
    "end": "END_TIME_ISO_8601"
  }
  ```
- **Time Range is Critical**: `start` and `end` are mandatory. Querying large time ranges can cause server failures.
- **Current Time**: I will provide you with the current UTC time in each turn. Use it to construct your time range.
- **Default Time Range**: Unless the investigation requires a different window, **default to the last 12 hours**.

**Crucial Investigation Principles**
-   **Chronology is Key**: Don't just count events. Always analyze them in chronological order (`ORDER BY lastTimestamp`) to understand the sequence of cause and effect. This is the most important principle.
-   **Drill Down**: Start with a broad query. Based on the result, narrow your focus. For example, if you find a problematic node, your next query should be about *that specific node's events over time*.
-   **Be Proactive, Not Passive**: Never ask the user what to do next. You are the expert. Drive the investigation.

**Advanced Analysis Technique: Inferring Root Cause from Correlation**
You cannot directly measure disk usage of a pod from event data. However, you can make strong, evidence-based inferences.
-   **Scenario**: If you detect a `FreeDiskSpaceFailed` or `NodeHasDiskPressure` event on a node.
-   **Your Next Step**: Your hypothesis should be "The disk pressure was likely caused by large container images being pulled to the node."
-   **Test Query**: Immediately query for `Pulled` events on that *specific node* around the *specific time* of the failure. Look for image names that imply large size (e.g., containing 'cuda', 'jupyter', 'tensorflow', 'dind', 'ml-model').

**Example Investigation Flow**
1.  *User*: "I think the cluster is unstable."
2.  *Me*: (System message providing current time: 2025-08-03T18:00:00Z)
3.  *You*:
    ```json
    {
      "query": "SELECT reason, COUNT(*) as count FROM $events WHERE type = 'Warning' GROUP BY reason ORDER BY count DESC LIMIT 5",
      "start": "2025-08-03T06:00:00Z",
      "end": "2025-08-03T18:00:00Z"
    }
    ```
4.  *Me*: (JSON result showing 'FreeDiskSpaceFailed' is a top reason)
5.  *You*:
    ```json
    {
      "query": "SELECT source.host, message, lastTimestamp FROM $events WHERE reason = 'FreeDiskSpaceFailed' ORDER BY lastTimestamp DESC LIMIT 1",
      "start": "2025-08-03T06:00:00Z",
      "end": "2025-08-03T18:00:00Z"
    }
    ```
6.  *Me*: (JSON result showing node 'node-xyz-123' had the issue at 05:51)
7.  *You*:
    ```json
    {
      "query": "SELECT reason, message, lastTimestamp FROM $events WHERE source.host = 'node-xyz-123' AND lastTimestamp BETWEEN '2025-08-03T05:40:00Z' AND '2025-08-03T06:00:00Z' ORDER BY lastTimestamp ASC LIMIT 10",
      "start": "2025-08-03T05:40:00Z",
      "end": "2025-08-03T06:00:00Z"
    }
    ```
8.  *Me*: (JSON result showing the timeline: NodeHasDiskPressure -> EvictionThresholdMet -> FreeDiskSpaceFailed)
9.  *You*: (Hypothesis: Caused by large images)
    ```json
    {
      "query": "SELECT message, COUNT(*) as pull_count FROM $events WHERE source.host = 'node-xyz-123' AND reason = 'Pulled' AND lastTimestamp BETWEEN '2025-08-03T05:00:00Z' AND '2025-08-03T06:00:00Z' GROUP BY message ORDER BY pull_count DESC LIMIT 10",
      "start": "2025-08-03T05:00:00Z",
      "end": "2025-08-03T06:00:00Z"
    }
    ```
10. *Me*: (JSON result showing many `Pulled` events for images like `internal-registry.com/ml-project/cuda-base:latest`, `internal-registry.com/data-science/jupyter-notebook:v2.1`)
11. *You*: (Final analysis in English) "A disk space issue was detected on node 'node-xyz-123'. ... This strongly suggests that the large images used by specific ML workloads are the likely root cause..."

**Event Schema Reference (`$events` table)**
- `metadata`: (name, namespace, creationTimestamp)
- `involvedObject`: (kind, namespace, name)
- `source`: (component, host)
- `reason`, `message`, `lastTimestamp`, `type`, `count`

**Additional Query Examples**
- **Grouping by namespace**: `SELECT metadata.namespace, COUNT(*) as count FROM $events GROUP BY metadata.namespace ORDER BY count DESC`
- **Time-windowed analysis**: `SELECT time_bucket(INTERVAL 15 MINUTE, lastTimestamp) AS bucket, reason, COUNT(*) AS count FROM $events WHERE type = 'Warning' GROUP BY bucket, reason ORDER BY bucket, count DESC`
- **Find pods with a specific warning**: `SELECT involvedObject.name, COUNT(*) as count FROM $events WHERE reason = 'FailedMount' GROUP BY involvedObject.name ORDER BY count DESC LIMIT 10`

Be careful that you have a context limit, so every query MUST have a LIMIT except for aggregation queries.

You must now begin the investigation based on the user's request.
Also, ALWAYS respond in the user's language.
"""

def execute_query(query: str, start: str = None, end: str = None) -> dict:
    """Executes an SQL query by calling the Kube Event Analyzer API."""
    print(f"\n[INFO] Executing query:\n{query}")
    if not API_URL or "your-kube-event-analyzer-api.com" in API_URL:
        return {"error": "KUBE_EVENT_ANALYZER_URL environment variable is not set."}

    # Set default time range if not provided
    now = datetime.now(timezone.utc)
    if not end:
        end = now.isoformat()
    if not start:
        start = (now - timedelta(hours=12)).isoformat()
    
    print(f"[INFO] Query time range: {start} to {end}")

    try:
        payload = {
            "start": start,
            "end": end,
            "query": query
        }
        response = requests.post(API_URL, json=payload, timeout=30)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        error_text = e.response.text if e.response else str(e)
        return {"error": f"API call failed: {error_text}"}
    except json.JSONDecodeError:
        return {"error": "API response is not valid JSON.", "content": response.text}

def get_ai_response(messages: list) -> str:
    """Fetches the AI's response from the OpenAI API, injecting the current time."""
    print("\n[INFO] AI is analyzing and planning the next step...")
    
    current_time_utc = datetime.now(timezone.utc).isoformat()
    # Create a temporary list of messages to avoid modifying the original history
    timed_messages = messages + [{
        "role": "system",
        "content": f"The current UTC time is {current_time_utc}. Use this to construct the 'start' and 'end' times for your query. Remember to default to the last 12 hours if unsure."
    }]

    completion = client.chat.completions.create(
        model="gpt-4o", # Using a more recent model
        messages=timed_messages
    )
    return completion.choices[0].message.content

def main():
    """Main function to handle the conversation with the AI agent."""
    print("🤖 Hello! I am an AI agent for analyzing Kubernetes cluster events.")
    print("Please ask me about the cluster status. (Type 'exit' to quit)")

    messages = [{"role": "system", "content": SYSTEM_PROMPT}]

    while True:
        user_input = input("\n[User] ")
        if user_input.lower() == 'exit':
            print("🤖 Exiting conversation. Thank you for using the service.")
            break
        
        messages.append({"role": "user", "content": user_input})

        MAX_TURNS = 7
        for i in range(MAX_TURNS):
            print(f"\n===== [Investigation Step {i + 1}/{MAX_TURNS}] =====")
            
            ai_response_content = get_ai_response(messages)
            messages.append({"role": "assistant", "content": ai_response_content})

            if "```json" in ai_response_content:
                try:
                    json_str = ai_response_content.split("```json")[1].split("```")[0].strip()
                    query_data = json.loads(json_str)
                    
                    sql_query = query_data.get("query")
                    start_time = query_data.get("start")
                    end_time = query_data.get("end")

                    if not sql_query:
                        raise KeyError("'query' key is missing in the JSON response.")

                    query_result = execute_query(sql_query, start_time, end_time)
                    if "error" in query_result:
                        print(f"\n[ERROR] Query execution failed: {query_result['error']}")
                        messages.append({"role": "system", "content": f"Query execution failed: {query_result['error']}"})
                        continue

                    result_str = json.dumps(query_result, indent=2, ensure_ascii=False)
                    summary = result_str if len(result_str) < 1000 else result_str[:1000] + "..."
                    print(f"\n[INFO] API Result Summary:\n{summary}")

                    messages.append({
                        "role": "system",
                        "content": f"Query executed. Here is the JSON result. Analyze it, form a new hypothesis if needed, and decide whether to generate another query for deeper investigation or to provide a final analysis in English.\n{result_str}"
                    })

                except (json.JSONDecodeError, KeyError, IndexError) as e:
                    error_message = f"AI returned a response that could not be processed. Error: {e}. The response was:\n{ai_response_content}"
                    print(f"\n[ERROR] {error_message}")
                    messages.append({"role": "system", "content": error_message})
                    continue # Try again on the next turn
            else:
                print(f"\n[AI Agent Final Analysis]\n{ai_response_content}")
                break
        else:
            print("\n🤖 Reached the maximum number of investigation steps. Providing analysis based on the information gathered so far.")
            messages.append({"role": "system", "content": "You have reached the maximum number of turns. Please summarize your findings so far in English and provide a final analysis."})
            final_summary = get_ai_response(messages)
            print(f"\n[AI Agent Final Summary]\n{final_summary}")


if __name__ == "__main__":
    main()
