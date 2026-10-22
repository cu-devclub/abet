package rabbitmq_test

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	amqp "github.com/rabbitmq/amqp091-go"

	"abet/main-service/internal/adapter/outbound/rabbitmq"
	"abet/main-service/internal/core/domain"
)

var _ = Describe("rabbitmq adapter", func() {
	It("builds an escaped AMQP URL that parses back to the same settings", func() {
		cfg := rabbitmq.Config{Host: "rabbitmq", Port: "5672", User: "abet", Pass: "p@ss/w:rd#1", VHost: "/"}
		uri, err := amqp.ParseURI(cfg.URL())
		Expect(err).NotTo(HaveOccurred())
		Expect(uri.Host).To(Equal("rabbitmq"))
		Expect(uri.Port).To(Equal(5672))
		Expect(uri.Username).To(Equal("abet"))
		Expect(uri.Password).To(Equal("p@ss/w:rd#1"))
		Expect(uri.Vhost).To(Equal("/"))

		cfg.VHost = "abet-vhost"
		uri, err = amqp.ParseURI(cfg.URL())
		Expect(err).NotTo(HaveOccurred())
		Expect(uri.Vhost).To(Equal("abet-vhost"))
	})

	It("escapes user and password with every special character", func() {
		const nasty = `p@ss:w/rd?#%&='"\ x`
		cfg := rabbitmq.Config{Host: "rabbitmq", Port: "5672", User: "u" + nasty, Pass: nasty, VHost: "/"}
		uri, err := amqp.ParseURI(cfg.URL())
		Expect(err).NotTo(HaveOccurred())
		Expect(uri.Host).To(Equal("rabbitmq"))
		Expect(uri.Username).To(Equal("u" + nasty))
		Expect(uri.Password).To(Equal(nasty))
	})

	It("falls back to port 5672 for a bad port", func() {
		uri, err := amqp.ParseURI(rabbitmq.Config{Host: "h", Port: "nope", User: "u", Pass: "p"}.URL())
		Expect(err).NotTo(HaveOccurred())
		Expect(uri.Port).To(Equal(5672))
	})

	It("backs off exponentially from 1s up to 30s", func() {
		Expect(rabbitmq.Backoff(0)).To(Equal(time.Second))
		Expect(rabbitmq.Backoff(1)).To(Equal(time.Second))
		Expect(rabbitmq.Backoff(2)).To(Equal(2 * time.Second))
		Expect(rabbitmq.Backoff(5)).To(Equal(16 * time.Second))
		Expect(rabbitmq.Backoff(6)).To(Equal(30 * time.Second))
		Expect(rabbitmq.Backoff(100)).To(Equal(30 * time.Second))
	})

	It("encodes the v2 job payload (and omits an absent cover)", func() {
		id := uuid.New()
		body, err := rabbitmq.EncodeJob(domain.BuildJob{
			JobID: id, ProgramName: "cs-2026", ZipKey: "uploads/a.zip",
			OutputPrefix: "outputs/a/", DeleteZipOnCompletion: true,
		})
		Expect(err).NotTo(HaveOccurred())
		var m map[string]any
		Expect(json.Unmarshal(body, &m)).To(Succeed())
		Expect(m).To(Equal(map[string]any{
			"job_id": id.String(), "program_name": "cs-2026", "storage": "s3", "zip_key": "uploads/a.zip",
			"output_prefix": "outputs/a/", "delete_zip_on_completion": true,
		}))

		body, _ = rabbitmq.EncodeJob(domain.BuildJob{JobID: id, CoverKey: "covers/c.pdf"})
		Expect(string(body)).To(ContainSubstring(`"cover_key":"covers/c.pdf"`))
		Expect(string(body)).NotTo(ContainSubstring("callback_url"))
		Expect(string(body)).NotTo(ContainSubstring("zip_path"))
	})

	It("decodes worker status messages", func() {
		ev, err := rabbitmq.DecodeStatus([]byte(`{"status":"success","job_id":"j1","program_name":"p","elapsed_seconds":3.2,"exported_pdfs":[],"exported_pdf_keys":["outputs/a/x-main.pdf"],"exported_report_keys":[],"timestamp":"t"}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(ev).To(Equal(domain.BuildStatusEvent{Status: "success", JobID: "j1", ExportedPDFKeys: []string{"outputs/a/x-main.pdf"}}))

		ev, err = rabbitmq.DecodeStatus([]byte(`{"status":"failed","job_id":"j2","error":"boom","traceback":"tb"}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(ev.Error).To(Equal("boom"))

		_, err = rabbitmq.DecodeStatus([]byte(`not json`))
		Expect(err).To(HaveOccurred())
	})
})
