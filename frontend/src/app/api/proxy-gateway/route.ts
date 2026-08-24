import { NextRequest, NextResponse } from 'next/server';

export async function GET(req: NextRequest) {
  const apiKey = req.headers.get('X-API-Key') || 'key_free_example';

  try {
    const res = await fetch('http://localhost/api/v1/search', {
      method: 'GET',
      headers: {
        'X-API-Key': apiKey,
      },
      cache: 'no-store',
    });

    const data = await res.json().catch(() => ({}));

    return NextResponse.json(data, {
      status: res.status,
      headers: {
        'X-Ratelimit-Limit': res.headers.get('X-Ratelimit-Limit') || '',
        'X-Ratelimit-Remaining': res.headers.get('X-Ratelimit-Remaining') || '',
        'X-Request-Id': res.headers.get('X-Request-Id') || '',
      },
    });
  } catch (error) {
    return NextResponse.json(
      { message: 'Live gateway fallback', error: String(error) },
      { status: 200 }
    );
  }
}
