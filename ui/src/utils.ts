
export async function getData(url: string): Promise<any> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Failed to fetch: ' + response.statusText);
  }
  return response.json();
}
