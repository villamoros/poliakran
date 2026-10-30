import { NextResponse, type NextRequest } from "next/server";

const SESSION_COOKIE = "pa_session";

// Comprobación optimista: sin cookie de sesión no tiene sentido mostrar el panel.
// La validación real de la sesión ocurre siempre en el backend.
export function proxy(request: NextRequest) {
  if (!request.cookies.has(SESSION_COOKIE)) {
    const url = new URL("/ingresar", request.url);
    url.searchParams.set("next", request.nextUrl.pathname);
    return NextResponse.redirect(url);
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/panel/:path*"],
};
