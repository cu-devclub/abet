FROM python:3.11-slim

# Prevent interactive prompts during installation
ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1

# Install system dependencies: TeX Live toolchain, Poppler (pdfinfo), Git, and Utilities
RUN apt-get update && apt-get install -y --no-install-recommends \
    texlive-latex-base \
    texlive-latex-recommended \
    texlive-latex-extra \
    texlive-fonts-recommended \
    lmodern \
    latexmk \
    poppler-utils \
    git \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy requirement definitions first for caching
COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

# Install Playwright browser dependencies (if LMS watch is used)
RUN python3 -m playwright install chromium --with-deps

# Copy the entire ABET engine and microservice source
COPY . /app

# Ensure default output directory exists
RUN mkdir -p /var/storage/output_pdfs /var/storage/uploads

# Run worker script
CMD ["python3", "-u", "worker.py"]
