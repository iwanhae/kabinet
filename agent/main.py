import os
import requests
import json
from openai import OpenAI
from datetime import datetime, timedelta, timezone
from dataclasses import dataclass, field

# --- Configuration ---
API_URL = os.getenv("KUBE_EVENT_ANALYZER_URL", "http://127.0.0.1:8080/query")
try:
    client = OpenAI(
        api_key=os.getenv("OPENAI_API_KEY"),
        base_url=os.getenv("OPENAI_API_BASE")
    )
    if not os.getenv("OPENAI_API_KEY"):
        raise ValueError("OPENAI_API_KEY environment variable is not set.")
except Exception as e:
    print(f"Failed to initialize OpenAI API client: {e}")
    exit(1)

# --- Enhanced System Prompt ---
SYSTEM_PROMPT = """
You are a proactive and autonomous expert AI assistant for troubleshooting Kubernetes cluster events.
Your primary goal is to independently investigate user issues by forming and testing hypotheses.

**Your Core Mission**
A user will state a problem. You will then take charge of the entire investigation.
1.  **Analyze & Hypothesize**: Based on the user's request and the data available, analyze the situation and form a clear, testable hypothesis about the root cause.
2.  **Query & Test**: Generate a single, precise SQL query to prove or disprove your current hypothesis.
3.  **Analyze & Iterate**: After I provide the JSON result for your query, analyze it.
    - If your hypothesis is confirmed and you have enough information, conclude the investigation.
    - If your hypothesis is disproven or you need more data, form a *new* hypothesis and generate the next query.
4.  **Conclude**: When the investigation is complete, provide a final, comprehensive analysis in the user's language.

**Crucial Response Format**
You MUST respond with a single JSON object. No other text or explanation.

**If you are continuing the investigation, use this JSON structure:**
```json
{
  "thought": "A brief, one-sentence rationale for your next action.",
  "hypothesis": "Your current, specific, testable hypothesis.",
  "query": {
    "sql": "SELECT ...",
    "start": "START_TIME_ISO_8601",
    "end": "END_TIME_ISO_8601"
  }
}
```

**If you are concluding the investigation, use this JSON structure:**
```json
{
  "thought": "The investigation is complete because I have identified the likely root cause.",
  "hypothesis": "The final, confirmed hypothesis.",
  "final_analysis": "Your comprehensive analysis and actionable recommendations for the user."
}
```

**How to Handle Query Results**
- **If the query returns results**: Analyze the data to see if it supports your hypothesis.
- **If the query returns an empty result `[]`**: This is a critical piece of information. It likely means your hypothesis was incorrect. You MUST state this in your 'thought' and formulate a new hypothesis.
- **Querying Time Range**: `start` and `end` are mandatory. Default to the last 12 hours unless the investigation requires a different window. I will provide the current UTC time.
- **Chronology is Key**: Use `ORDER BY lastTimestamp` to understand cause and effect.
- **Drill Down**: Start broad, then narrow your focus based on results (e.g., from cluster-wide warnings to a specific node's event timeline).

**Example Investigation Flows**

**Example 1: Analyzing `OutOfcpu` Failures**

1.  **User**: "We are seeing `OutOfcpu` errors, but I don't understand why. Pods should not be scheduled on nodes with insufficient CPU in the first place."
2.  **System**: (Provides current time: 2025-08-05T15:00:00Z)
3.  **AI**:
    ```json
    {
      "thought": "The user is right, this suggests a timing issue between the scheduler and kubelet. I'll start by finding a specific Pod that failed with 'OutOfcpu' to trace its full event lifecycle.",
      "hypothesis": "The 'OutOfcpu' event is preceded by a 'FailedScheduling' event for the same pod, indicating a race condition or rapid change in node resource availability.",
      "query": {
        "sql": "SELECT involvedObject.name, source.host, message FROM $events WHERE reason = 'OutOfcpu' ORDER BY lastTimestamp DESC LIMIT 1",
        "start": "2025-08-05T03:00:00Z",
        "end": "2025-08-05T15:00:00Z"
      }
    }
    ```
4.  **System**: (Returns a summary: "Query returned 1 row... First row summary: {\"involvedObject.name\": \"app-migration-pod-abc123\", \"source.host\": \"worker-node-001\", ...}")
5.  **AI**:
    ```json
    {
      "thought": "Okay, the pod 'app-migration-pod-abc123' failed on 'worker-node-001'. Now I must trace all events for this specific pod to see what happened *before* the 'OutOfcpu' event.",
      "hypothesis": "A 'FailedScheduling' event for pod 'app-migration-pod-abc123' occurred shortly before the 'OutOfcpu' event, which will confirm the race condition theory.",
      "query": {
        "sql": "SELECT lastTimestamp, reason, type, source.host, message FROM $events WHERE involvedObject.name = 'app-migration-pod-abc123' ORDER BY lastTimestamp ASC",
        "start": "2025-08-05T03:00:00Z",
        "end": "2025-08-05T15:00:00Z"
      }
    }
    ```
6.  **System**: (Returns a summary: "Query returned 2 rows... First event is 'FailedScheduling' with message '...2 Insufficient cpu...'. Second event is 'OutOfcpu' on 'worker-node-001'.")
7.  **AI**:
    ```json
    {
      "thought": "The investigation is complete. The event timeline clearly shows 'FailedScheduling' followed by 'OutOfcpu', which confirms the hypothesis.",
      "hypothesis": "The 'OutOfcpu' error is caused by a race condition where a node's available resources change between the time the scheduler makes a decision and the time the kubelet on the node attempts to admit the pod.",
      "final_analysis": "The 'OutOfcpu' events are not a simple scheduling failure, but a symptom of a race condition on nodes with high resource churn. The scheduler initially saw the node as having insufficient CPU ('FailedScheduling'). A moment later, a pod likely terminated, freeing up just enough CPU for the scheduler to succeed. However, by the time the pod was sent to the node's kubelet, other pods had consumed that sliver of CPU, causing the kubelet's admission check to fail. This indicates the node is running 'hot' with very little CPU headroom. Recommend reviewing resource requests/limits for pods on these nodes and consider using pod anti-affinity to spread workloads more evenly."
    }
    ```

**Example 2: Analyzing Disk Pressure Failures**

1.  **User**: "I think the cluster is unstable."
2.  **System**: (Provides current time: 2025-08-03T18:00:00Z)
3.  **AI**:
    ```json
    {
      "thought": "The user has a vague concern. I will start by checking the overall health of the cluster by looking at the top warning events in the last 12 hours.",
      "hypothesis": "The cluster is experiencing a high number of a specific type of warning event.",
      "query": {
        "sql": "SELECT reason, COUNT(*) as count FROM $events WHERE type = 'Warning' GROUP BY reason ORDER BY count DESC LIMIT 5",
        "start": "2025-08-03T06:00:00Z",
        "end": "2025-08-03T18:00:00Z"
      }
    }
    ```
4.  **System**: (Returns a summary: "Query returned 5 rows. Columns: reason, count. First row summary: {\"reason\": \"FailedMount\", \"count\": 1234}")
5.  **AI**:
    ```json
    {
      "thought": "The most frequent warning is 'FailedMount'. I need to find out which node is affected to understand the scope of the problem.",
      "hypothesis": "A specific node is repeatedly failing to mount volumes, causing cluster instability.",
      "query": {
        "sql": "SELECT source.host, message, lastTimestamp FROM $events WHERE reason = 'FailedMount' ORDER BY lastTimestamp DESC LIMIT 1",
        "start": "2025-08-03T06:00:00Z",
        "end": "2025-08-03T18:00:00Z"
      }
    }
    ```
6.  **System**: (Returns a summary: "Query returned 1 row... First row summary: {\"source.host\": \"worker-node-002\", ...}")
7.  **AI**:
    ```json
    {
      "thought": "The issue is located on 'worker-node-002'. Now I need to analyze the sequence of events on that specific node around the time of the failure to find the root cause.",
      "hypothesis": "The 'FailedMount' event on 'worker-node-002' is preceded by other system-level warnings like disk or memory pressure.",
      "query": {
        "sql": "SELECT reason, message, lastTimestamp FROM $events WHERE source.host = 'worker-node-002' ORDER BY lastTimestamp ASC LIMIT 10",
        "start": "2025-08-03T17:30:00Z",
        "end": "2025-08-03T18:00:00Z"
      }
    }
    ```
8.  **System**: (Returns a summary showing a sequence of events like 'NodeHasDiskPressure' -> 'EvictionThresholdMet' -> 'FailedMount')
9.  **AI**:
    ```json
    {
      "thought": "The investigation is complete. I have found a clear causal link between disk pressure and the mount failures.",
      "hypothesis": "The 'FailedMount' errors on 'worker-node-002' are the result of the node running out of disk space, which prevented new volumes from being mounted.",
      "final_analysis": "The cluster instability is caused by 'FailedMount' events concentrated on node 'worker-node-002'. The timeline analysis reveals that these failures are a direct symptom of 'NodeHasDiskPressure' events. This indicates the node is running out of disk space. Recommend inspecting the disk usage on 'worker-node-002' and cleaning up unnecessary files, such as old container images or logs."
    }
    ```

You must now begin the investigation based on the user's request.
"""

@dataclass
class InvestigationState:
    """Manages the state of a single investigation."""
    user_request: str
    max_turns: int = 7
    turn: int = 0
    hypothesis: str = "Initial hypothesis pending."
    messages: list = field(default_factory=list)

    def __post_init__(self):
        self.messages.append({"role": "system", "content": SYSTEM_PROMPT})
        self.messages.append({"role": "user", "content": self.user_request})

    def add_ai_response(self, response: dict):
        self.messages.append({"role": "assistant", "content": json.dumps(response)})

    def add_system_observation(self, content: str):
        print(f"\n[SYSTEM] {content}")
        self.messages.append({"role": "system", "content": content})

    def is_complete(self) -> bool:
        return self.turn >= self.max_turns

def execute_query(query: str, start: str, end: str) -> dict:
    """Executes an SQL query against the Kube Event Analyzer API."""
    print(f"\n[INFO] Executing query:\n{query}")
    print(f"[INFO] Query time range: {start} to {end}")
    try:
        payload = {"start": start, "end": end, "query": query}
        response = requests.post(API_URL, json=payload, timeout=30)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        error_text = e.response.text if e.response else str(e)
        return {"error": f"API call failed: {error_text}"}
    except json.JSONDecodeError:
        return {"error": "API response is not valid JSON.", "content": response.text}

def get_ai_plan(state: InvestigationState) -> dict:
    """Fetches the AI's next plan of action (thought, hypothesis, query)."""
    print("\n[INFO] AI is analyzing and planning the next step...")
    
    current_time_utc = datetime.now(timezone.utc).isoformat()
    timed_messages = state.messages + [{
        "role": "system",
        "content": f"The current UTC time is {current_time_utc}. Use this to construct your query's time range."
    }]

    completion = client.chat.completions.create(
        model="gpt-4.1",
        messages=timed_messages,
        response_format={"type": "json_object"}
    )
    
    response_content = completion.choices[0].message.content
    try:
        return json.loads(response_content)
    except json.JSONDecodeError:
        print(f"[ERROR] AI returned invalid JSON:\n{response_content}")
        return {"final_analysis": "Error: The AI returned a response that was not valid JSON."}


def summarize_result(result: dict) -> str:
    """Creates a concise summary of a query result to save context space."""
    if "error" in result:
        return f"Query failed with error: {result['error']}"
    
    results_list = result.get("results", [])
    if not results_list:
        return "Query returned no results (an empty list: [])."

    count = len(results_list)
    columns = list(results_list[0].keys()) if results_list else []
    
    summary = f"Query returned {count} rows. Columns: {', '.join(columns)}. "
    if count > 0:
        summary += f"First row summary: {json.dumps(results_list[0])}"
    
    return summary


def main():
    """Main function to handle the conversation with the AI agent."""
    print("🤖 Hello! I am an AI agent for analyzing Kubernetes cluster events.")
    print("Please state the problem you are observing. (Type 'exit' to quit)")

    while True:
        user_input = input("\n[User] ")
        if user_input.lower() == 'exit':
            print("🤖 Exiting conversation.")
            break

        state = InvestigationState(user_request=user_input)

        while not state.is_complete():
            state.turn += 1
            print(f"\n===== [Investigation Step {state.turn}/{state.max_turns}] =====")
            
            ai_plan = get_ai_plan(state)
            state.add_ai_response(ai_plan)

            print(f"[AI Thought] {ai_plan.get('thought')}")
            state.hypothesis = ai_plan.get('hypothesis', state.hypothesis)
            print(f"[AI Hypothesis] {state.hypothesis}")

            if "final_analysis" in ai_plan:
                print(f"\n[AI Agent Final Analysis]\n{ai_plan['final_analysis']}")
                break

            query_info = ai_plan.get("query")
            if not query_info or not query_info.get("sql"):
                state.add_system_observation("AI did not provide a valid query. Ending investigation.")
                print("\n[AI Agent Final Analysis]\nNo further action taken.")
                break

            query_result = execute_query(
                query_info["sql"],
                query_info["start"],
                query_info["end"]
            )
            
            result_summary = summarize_result(query_result)
            system_observation = f"Query executed. Here is a summary of the result:\n{result_summary}"
            state.add_system_observation(system_observation)

        else:
            print("\n🤖 Reached the maximum number of investigation steps.")
            state.add_system_observation("You have reached the maximum number of turns. Please summarize your findings and provide a final analysis.")
            final_plan = get_ai_plan(state)
            print(f"\n[AI Agent Final Summary]\n{final_plan.get('final_analysis', 'No summary provided.')}")


if __name__ == "__main__":
    main()
