import os
from datetime import datetime, timezone
from fastapi import FastAPI, Header, HTTPException, Depends
from pydantic import BaseModel, Field
from sqlalchemy import (
    create_engine, Column, Integer, String, DateTime, Text, func, desc, Float
)
from sqlalchemy.orm import declarative_base, sessionmaker, Session

# 兼容 SQLite 和 Postgres
DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "sqlite:///./data/geocache.db"
)
API_KEYS = [k.strip() for k in os.getenv("API_KEYS", "change_me_api_key").split(",") if k.strip()]

# SQLite 特殊配置
if DATABASE_URL.startswith("sqlite"):
    engine = create_engine(DATABASE_URL, connect_args={"check_same_thread": False})
else:
    engine = create_engine(DATABASE_URL, pool_pre_ping=True)

SessionLocal = sessionmaker(bind=engine, autocommit=False, autoflush=False)
Base = declarative_base()


class IpReport(Base):
    __tablename__ = "ip_reports"
    id = Column(Integer, primary_key=True, index=True)
    ip = Column(String(64), index=True, nullable=False)
    # 归属地详细字段
    location = Column(String(256), nullable=True, default="")  # 位置（IP归属地）
    district = Column(String(128), nullable=True, default="")  # 区
    street = Column(String(256), nullable=True, default="")   # 街道
    isp = Column(String(128), nullable=True, default="")      # 网络服务商
    latitude = Column(Float, nullable=True)                  # 纬度
    longitude = Column(Float, nullable=True)                 # 经度
    # 其他字段
    provider = Column(String(64), nullable=False, default="unknown")
    client_version = Column(String(64), nullable=False, default="")
    created_at = Column(DateTime(timezone=True), nullable=False, default=lambda: datetime.now(timezone.utc))


class IpBest(Base):
    __tablename__ = "ip_best"
    id = Column(Integer, primary_key=True, index=True)
    ip = Column(String(64), unique=True, index=True, nullable=False)
    # 归属地详细字段
    location = Column(String(256), nullable=True, default="")  # 位置（IP归属地）
    district = Column(String(128), nullable=True, default="")  # 区
    street = Column(String(256), nullable=True, default="")   # 街道
    isp = Column(String(128), nullable=True, default="")      # 网络服务商
    latitude = Column(Float, nullable=True)                  # 纬度
    longitude = Column(Float, nullable=True)                 # 经度
    # 其他字段
    provider = Column(String(64), nullable=False, default="unknown")
    count = Column(Integer, nullable=False, default=1)
    updated_at = Column(DateTime(timezone=True), nullable=False, default=lambda: datetime.now(timezone.utc))


Base.metadata.create_all(bind=engine)


class ReportIn(BaseModel):
    ip: str = Field(min_length=1, max_length=64)
    # 详细归属地字段（可选）
    location: str | None = Field(default=None, max_length=256)  # 位置
    district: str | None = Field(default=None, max_length=128)  # 区
    street: str | None = Field(default=None, max_length=256)   # 街道
    isp: str | None = Field(default=None, max_length=128)      # 网络服务商
    latitude: float | None = Field(default=None)               # 纬度
    longitude: float | None = Field(default=None)              # 经度
    # 其他字段
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


app = FastAPI(title="GeoCache", version="0.1.0")


@app.get("/healthz")
def healthz():
    return {"ok": True, "service": "GeoCache"}


@app.post("/v1/ip/report", dependencies=[Depends(auth)])
def report_ip(payload: ReportIn, db: Session = Depends(get_db)):
    db.add(IpReport(
        ip=payload.ip,
        location=payload.location or "",
        district=payload.district or "",
        street=payload.street or "",
        isp=payload.isp or "",
        latitude=payload.latitude,
        longitude=payload.longitude,
        provider=payload.provider,
        client_version=payload.client_version,
    ))
    db.commit()

    # 简单聚合：同 ip + location + isp 组合，选出现次数最多的作为 best
    from sqlalchemy import and_
    q = (
        db.query(
            IpReport.ip,
            IpReport.location,
            IpReport.district,
            IpReport.street,
            IpReport.isp,
            IpReport.latitude,
            IpReport.longitude,
            IpReport.provider,
            func.count(IpReport.id).label("c"),
        )
        .filter(IpReport.ip == payload.ip)
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
        .order_by(desc("c"))
        .first()
    )

    if q:
        best = db.query(IpBest).filter(IpBest.ip == payload.ip).first()
        if best:
            best.location = q.location or ""
            best.district = q.district or ""
            best.street = q.street or ""
            best.isp = q.isp or ""
            best.latitude = q.latitude
            best.longitude = q.longitude
            best.provider = q.provider
            best.count = q.c
            best.updated_at = datetime.now(timezone.utc)
        else:
            best = IpBest(
                ip=q.ip,
                location=q.location or "",
                district=q.district or "",
                street=q.street or "",
                isp=q.isp or "",
                latitude=q.latitude,
                longitude=q.longitude,
                provider=q.provider,
                count=q.c,
                updated_at=datetime.now(timezone.utc)
            )
            db.add(best)
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
        "updated_at": best.updated_at.isoformat(),
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)