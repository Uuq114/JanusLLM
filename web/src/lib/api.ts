export type AdminCredentials = {
  username: string;
  password: string;
};

const API_BASE_URL = import.meta.env.VITE_JANUS_API_BASE_URL ?? "";

type ApiEnvelope<T> = {
  data?: T;
};

async function requestJson<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init.headers
    }
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with ${response.status}`);
  }

  return response.json() as Promise<T>;
}

function basicAuth({ username, password }: AdminCredentials): string {
  return `Basic ${window.btoa(`${username}:${password}`)}`;
}

export async function listAdminResource<T>(
  resource: "organizations" | "teams" | "keys",
  credentials: AdminCredentials
): Promise<T[]> {
  const envelope = await requestJson<ApiEnvelope<T[]>>(`/v1/admin/${resource}`, {
    headers: {
      Authorization: basicAuth(credentials)
    }
  });
  return envelope.data ?? [];
}

export async function listAccessibleModels(apiKey: string): Promise<string[]> {
  const envelope = await requestJson<ApiEnvelope<Array<{ id: string }>>>("/v1/models", {
    headers: {
      Authorization: `Bearer ${apiKey}`
    }
  });
  return (envelope.data ?? []).map((model) => model.id);
}
