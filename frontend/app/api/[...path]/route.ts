import { NextRequest, NextResponse } from 'next/server';

export const dynamic = 'force-dynamic';
export const runtime = 'nodejs';
export const maxDuration = 120;

const BACKEND_URL = process.env.BACKEND_URL || 'http://127.0.0.1:8080';

async function proxyRequest(req: NextRequest, path: string[]) {
  const target = `${BACKEND_URL}/api/${path.join('/')}${req.nextUrl.search}`;
  const headers = new Headers(req.headers);
  headers.delete('host');
  headers.delete('connection');
  headers.delete('content-length');

  const init: RequestInit = {
    method: req.method,
    headers,
    redirect: 'manual',
    cache: 'no-store',
  };

  if (req.method !== 'GET' && req.method !== 'HEAD') {
    init.body = await req.arrayBuffer();
  }

  try {
    const res = await fetch(target, init);
    const outHeaders = new Headers(res.headers);
    outHeaders.delete('transfer-encoding');
    return new Response(res.body, {
      status: res.status,
      statusText: res.statusText,
      headers: outHeaders,
    });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'socket hang up';
    return NextResponse.json(
      {
        error:
          'Backend is not reachable on port 8080. Start it with `go run main.go` in the backend folder, then try again.',
        detail: message,
      },
      { status: 502 }
    );
  }
}

type RouteContext = { params: { path: string[] } };

export async function GET(req: NextRequest, ctx: RouteContext) {
  return proxyRequest(req, ctx.params.path);
}

export async function POST(req: NextRequest, ctx: RouteContext) {
  return proxyRequest(req, ctx.params.path);
}

export async function PUT(req: NextRequest, ctx: RouteContext) {
  return proxyRequest(req, ctx.params.path);
}

export async function PATCH(req: NextRequest, ctx: RouteContext) {
  return proxyRequest(req, ctx.params.path);
}

export async function DELETE(req: NextRequest, ctx: RouteContext) {
  return proxyRequest(req, ctx.params.path);
}

export async function OPTIONS() {
  return new Response(null, { status: 204 });
}
