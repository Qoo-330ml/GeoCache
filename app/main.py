import os
from datetime import datetime, timedelta, timezone

from fastapi import Depends, FastAPI, Header, HTTPException
from pydantic import BaseModel, Field
from sqlalchemy import (
    Column,
    DateTime,
    Float,
    Index,
    Integer,
    String,
    case,
    create_engine,
    func,
)
from sqlalchemy.orm import Session, declarative_base, sessionmaker


def parse_provider_weights(raw: str) -> dict[str, int]:
    weights: dict[str, int] = {}
    for item in raw.split(","):
        item = item.strip()
        if not item or ":" not in item:
            continue
        provider, weight = item.split(":", 1)
        provider = provider.strip().lower()
        if not provider:
            continue
        try:
            weights[provider] = int(weight.strip())
        except ValueError:
            continue
    return weights


def normalize_provider(provider: str | None) -> str:
    return (provider or "").strip().lower()


APP_TIMEZONE = timezone(timedelta(hours=8), name="Asia/Shanghai")


def now_in_app_timezone() -> datetime:
    return datetime.now(APP_TIMEZONE)


def ensure_app_timezone(value: datetime | None) -> datetime | None:
    if value is None:
        return None
    if value.tzinfo is None:
        return value.replace(tzinfo=APP_TIMEZONE)
    return value.astimezone(APP_TIMEZONE)


DATABASE_URL = os.getenv("DATABASE_URL", "postgresql+psycopg://geocache:geocache@geocache-db:5432/geocache")
API_KEYS = [k.strip() for k in os.getenv("API_KEYS", "change_me_api_key").split(",") if k.strip()]
PROVIDER_WEIGHT_OVERRIDES = parse_provider_weights(os.getenv("PROVIDER_WEIGHTS", ""))

if DATABASE_URL.startswith("sqlite"):
    engine = create_engine(DATABASE_URL, connect_args={"check_same_thread": False})
else:
    engine = create_engine(DATABASE_URL, pool_pre_ping=True)

SessionLocal = sessionmaker(bind=engine, autocommit=False, autoflush=False)
Base = declarative_base()


class IpReport(Base):
    __tablename__ = "ip_reports"
    __table_args__ = (Index("ix_ip_reports_ip_created_at", "ip", "created_at"),)

    id = Column(Integer, primary_key=True, index=True)
    ip = Column(String(64), index=True, nullable=False)
    location = Column(String(256), nullable=True, default="")
    district = Column(String(128), nullable=True, default="")
    street = Column(String(256), nullable=True, default="")
    isp = Column(String(128), nullable=True, default="")
    latitude = Column(Float, nullable=True)
    longitude = Column(Float, nullable=True)
    provider = Column(String(64), nullable=False, default="unknown")
    client_version = Column(String(64), nullable=False, default="")
    created_at = Column(DateTime(timezone=True), nullable=False, default=now_in_app_timezone)


class IpBest(Base):
    __tablename__ = "ip_best"

    id = Column(Integer, primary_key=True, index=True)
    ip = Column(String(64), unique=True, index=True, nullable=False)
    location = Column(String(256), nullable=True, default="")
    district = Column(String(128), nullable=True, default="")
    street = Column(String(256), nullable=True, default="")
    isp = Column(String(128), nullable=True, default="")
    latitude = Column(Float, nullable=True)
    longitude = Column(Float, nullable=True)
    provider = Column(String(64), nullable=False, default="unknown")
    count = Column(Integer, nullable=False, default=1)
    updated_at = Column(DateTime(timezone=True), nullable=False, default=now_in_app_timezone)


Base.metadata.create_all(bind=engine)


class ReportIn(BaseModel):
    ip: str = Field(min_length=1, max_length=64)
    location: str | None = Field(default=None, max_length=256)
    district: str | None = Field(default=None, max_length=128)
    street: str | None = Field(default=None, max_length=256)
    isp: str | None = Field(default=None, max_length=128)
    latitude: float | None = Field(default=None)
    longitude: float | None = Field(default=None)
    provider: str = Field(default="unknown", max_length=64)
    client_version: str = Field(default="", max_length=64)


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


def auth(x_api_key: str = Header(default="")):
    if x_api_key not in API_KEYS:
        raise HTTPException(status_code=401, detail="invalid api key")


def normalize_geo_fields(location, district, street, isp, latitude, longitude):
    return {
        "location": location or "",
        "district": district or "",
        "street": street or "",
        "isp": isp or "",
        "latitude": latitude,
        "longitude": longitude,
    }


def build_text_presence_expr(column):
    return case((func.length(func.trim(func.coalesce(column, ""))) > 0, 1), else_=0)


def build_value_presence_expr(column):
    return case((column.is_not(None), 1), else_=0)


def build_completeness_expr():
    return (
        build_text_presence_expr(IpReport.location)
        + build_text_presence_expr(IpReport.district)
        + build_text_presence_expr(IpReport.street)
        + build_text_presence_expr(IpReport.isp)
        + build_value_presence_expr(IpReport.latitude)
        + build_value_presence_expr(IpReport.longitude)
    )


def build_provider_weight_expr():
    normalized_provider = func.lower(func.trim(func.coalesce(IpReport.provider, "")))
    whens = [(normalized_provider == provider, weight) for provider, weight in PROVIDER_WEIGHT_OVERRIDES.items()]
    whens.append((normalized_provider.in_(["", "unknown"]), 0))
    return case(*whens, else_=1)


def build_report_match_clause(column, value):
    if value is None:
        return column.is_(None)
    return column == value


def get_best_ip_group(db: Session, ip: str):
    count_expr = func.count(IpReport.id).label("c")
    completeness_expr = func.max(build_completeness_expr()).label("completeness_score")
    provider_weight_expr = func.max(build_provider_weight_expr()).label("provider_weight")
    latest_created_at_expr = func.max(IpReport.created_at).label("latest_created_at")

    return (
        db.query(
            IpReport.ip,
            IpReport.location,
            IpReport.district,
            IpReport.street,
            IpReport.isp,
            IpReport.latitude,
            IpReport.longitude,
            IpReport.provider,
            count_expr,
            completeness_expr,
            provider_weight_expr,
            latest_created_at_expr,
        )
        .filter(IpReport.ip == ip)
        .group_by(
            IpReport.ip,
            IpReport.location,
            IpReport.district,
            IpReport.street,
            IpReport.isp,
            IpReport.latitude,
            IpReport.longitude,
            IpReport.provider,
        )
        .order_by(
            count_expr.desc(),
            completeness_expr.desc(),
            provider_weight_expr.desc(),
            latest_created_at_expr.desc(),
        )
        .first()
    )


def get_best_ip_report(db: Session, ip: str):
    group = get_best_ip_group(db, ip)
    if not group:
        return None, 0

    report = (
        db.query(IpReport)
        .filter(
            build_report_match_clause(IpReport.ip, group.ip),
            build_report_match_clause(IpReport.location, group.location),
            build_report_match_clause(IpReport.district, group.district),
            build_report_match_clause(IpReport.street, group.street),
            build_report_match_clause(IpReport.isp, group.isp),
            build_report_match_clause(IpReport.latitude, group.latitude),
            build_report_match_clause(IpReport.longitude, group.longitude),
            build_report_match_clause(IpReport.provider, group.provider),
        )
        .order_by(IpReport.created_at.desc(), IpReport.id.desc())
        .first()
    )
    if not report:
        return None, 0

    return report, group.c


def build_best_payload(report: IpReport, count: int):
    return {
        "ip": report.ip,
        **normalize_geo_fields(
            report.location,
            report.district,
            report.street,
            report.isp,
            report.latitude,
            report.longitude,
        ),
        "provider": report.provider or "unknown",
        "count": count,
        "updated_at": now_in_app_timezone(),
    }


def upsert_ip_best(db: Session, report: IpReport, count: int):
    best_payload = build_best_payload(report, count)
    best = db.query(IpBest).filter(IpBest.ip == report.ip).first()

    if best:
        for key, value in best_payload.items():
            setattr(best, key, value)
    else:
        db.add(IpBest(**best_payload))


app = FastAPI(title="GeoCache", version="0.1.0")


@app.get("/healthz")
def healthz():
    return {"ok": True, "service": "GeoCache"}


@app.post("/v1/ip/report", dependencies=[Depends(auth)])
def report_ip(payload: ReportIn, db: Session = Depends(get_db)):
    db.add(
        IpReport(
            ip=payload.ip,
            **normalize_geo_fields(
                payload.location,
                payload.district,
                payload.street,
                payload.isp,
                payload.latitude,
                payload.longitude,
            ),
            provider=payload.provider,
            client_version=payload.client_version,
        )
    )
    db.commit()

    report, count = get_best_ip_report(db, payload.ip)
    if report:
        upsert_ip_best(db, report, count)
        db.commit()

    return {"ok": True}


@app.get("/v1/ip/lookup")
def lookup_ip(ip: str, db: Session = Depends(get_db)):
    best = db.query(IpBest).filter(IpBest.ip == ip).first()
    if not best:
        return {"found": False, "ip": ip}
    return {
        "found": True,
        "ip": best.ip,
        "location": best.location,
        "district": best.district,
        "street": best.street,
        "isp": best.isp,
        "latitude": best.latitude,
        "longitude": best.longitude,
        "provider": best.provider,
        "count": best.count,
        "updated_at": ensure_app_timezone(best.updated_at).isoformat(),
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8080)
