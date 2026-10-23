from fastapi import APIRouter, Request, HTTPException
import stripe
import os

router = APIRouter()

# Initialize Stripe with test key (placeholder)
stripe.api_key = os.getenv("STRIPE_SECRET_KEY", "sk_test_dummy")


@router.post("/checkout")
async def create_checkout_session(request: Request):
    data = await request.json()
    # Expect amount and currency in request body; using placeholder values if missing
    amount = data.get("amount", 5000)  # amount in cents
    currency = data.get("currency", "usd")
    try:
        session = stripe.checkout.Session.create(
            payment_method_types=["card"],
            line_items=[
                {
                    "price_data": {
                        "currency": currency,
                        "product_data": {"name": "Devproxy SaaS Subscription"},
                        "unit_amount": amount,
                    },
                    "quantity": 1,
                }
            ],
            mode="payment",
            success_url="https://example.com/success?session_id={CHECKOUT_SESSION_ID}",
            cancel_url="https://example.com/cancel",
        )
        return {"checkout_url": session.url}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.post("/webhooks/stripe")
async def stripe_webhook(request: Request):
    payload = await request.body()
    sig_header = request.headers.get("stripe-signature")
    endpoint_secret = os.getenv("STRIPE_WEBHOOK_SECRET", "whsec_testsecret")
    try:
        event = stripe.Webhook.construct_event(payload, sig_header, endpoint_secret)
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))
    # Placeholder handling: just log event type
    return {"status": "received", "type": event["type"]}
