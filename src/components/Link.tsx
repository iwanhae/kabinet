import { useSearch, Link as WouterLink } from "wouter";

interface LinkProps {
  page: "" | "insight" | "discover";
  params: {
    where?: string;
    from?: string;
    to?: string;
    [key: string]: string | undefined;
  };
  children: React.ReactNode;
  onClick?: () => void;
  style?: React.CSSProperties;
}

export const Link = ({ page, params, children, onClick, style }: LinkProps) => {
  const search = useSearch();
  const searchParams = new URLSearchParams(search);
  const fromParam = searchParams.get("from") || "now-30m";
  const toParam = searchParams.get("to") || "now";

  if (page === "insight") page = "";

  const base = page ? `/p/${page}` : "/";
  const href = `${base}?${new URLSearchParams({
    from: fromParam,
    to: toParam,
    ...params, // params will override the search params
  }).toString()}`;

  return (
    <WouterLink href={href} onClick={onClick} style={style}>
      {children}
    </WouterLink>
  );
};
