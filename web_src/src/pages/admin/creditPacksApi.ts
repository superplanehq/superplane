export type CreditPackInput = {
  name: string;
  amount_cents: number;
  enabled: boolean;
  polar_product_id: string | null;
};

export type CreditPack = CreditPackInput & { id: string };
