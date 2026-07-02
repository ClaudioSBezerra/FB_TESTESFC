package handlers

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// Constantes de limite para upload de XMLs.
// Cópia tal qual de FB_APU04/backend/handlers/xml_upload.go — já auditadas
// (T-02-01 anti-ZIP-bomb, T-02-02 anti-path-traversal).
// ---------------------------------------------------------------------------

const (
	MaxUploadFileBytes   = 5 * 1024 * 1024 * 1024  // 5 GB — tamanho máximo do .zip/.xml enviado
	MaxUncompressedBytes = 20 * 1024 * 1024 * 1024 // 20 GB — proteção anti-ZIP bomb (total descomprimido)
	MaxSingleXMLBytes    = 10 * 1024 * 1024        // 10 MB — limite por XML individual
)

// namedXML representa um arquivo XML com seu nome de origem.
type namedXML struct {
	Name string
	Data []byte
}

// jsonErr escreve uma resposta de erro JSON padronizada.
func jsonErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ---------------------------------------------------------------------------
// extractXMLsFromZipFile abre um ZIP do disco e extrai os XMLs sem carregar
// o arquivo inteiro na RAM.
// Mitigações: T-02-01 (UncompressedSize64 acumulado, anti-ZIP-bomb),
//             T-02-02 (filepath.Base, anti path-traversal).
// ---------------------------------------------------------------------------

func extractXMLsFromZipFile(path string) ([]namedXML, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("ZIP inválido: %w", err)
	}
	defer r.Close()
	return extractXMLsFromZipFiles(r.File)
}

func extractXMLsFromZipFiles(files []*zip.File) ([]namedXML, error) {
	var totalUncompressed uint64
	var xmlFiles []namedXML

	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}
		// T-02-02: ignorar entries com path traversal
		if strings.Contains(f.Name, "..") {
			continue
		}
		baseName := filepath.Base(f.Name)
		if !strings.EqualFold(filepath.Ext(baseName), ".xml") {
			continue
		}

		// T-02-01: verificar tamanho acumulado antes de abrir (anti-ZIP bomb)
		totalUncompressed += f.UncompressedSize64
		if totalUncompressed > MaxUncompressedBytes {
			return nil, fmt.Errorf("conteúdo do ZIP excede limite de 20GB após descompressão")
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("erro ao abrir %s no ZIP: %w", baseName, err)
		}
		xmlData, err := io.ReadAll(io.LimitReader(rc, MaxSingleXMLBytes+1))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("erro ao ler %s no ZIP: %w", baseName, err)
		}
		if int64(len(xmlData)) > MaxSingleXMLBytes {
			return nil, fmt.Errorf("arquivo %s excede limite de 10MB por XML", baseName)
		}

		xmlFiles = append(xmlFiles, namedXML{Name: baseName, Data: xmlData})
	}

	return xmlFiles, nil
}

// ---------------------------------------------------------------------------
// xmlUploadError representa o erro de processamento de um XML individual.
// ---------------------------------------------------------------------------

type xmlUploadError struct {
	Arquivo string `json:"arquivo"`
	Erro    string `json:"erro"`
}

type xmlUploadResult struct {
	Importados int              `json:"importados"`
	Rejeitados int              `json:"rejeitados"`
	Total      int              `json:"total"`
	Erros      []xmlUploadError `json:"erros"`
}

// ---------------------------------------------------------------------------
// processSingleXML processa um único XML de NF-e de saída e persiste no banco.
// Isolamento de erro por arquivo: um erro aqui nunca aborta o lote inteiro
// (ver XMLUploadHandler — o chamador continua o loop após o erro).
// ---------------------------------------------------------------------------

func processSingleXML(db *sql.DB, companyID string, xf namedXML) error {
	data := xf.Data

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("arquivo vazio")
	}
	checkLen := 300
	if len(trimmed) < checkLen {
		checkLen = len(trimmed)
	}
	head := trimmed[:checkLen]
	if bytes.Contains(head, []byte("procCancNFe")) {
		return fmt.Errorf("evento de cancelamento — não é um documento fiscal")
	}

	proc, err := parseNFeXML(data)
	if err != nil {
		return err
	}

	inf := proc.NFe.InfNFe
	mod := strings.TrimSpace(inf.Ide.Mod)

	// Validação de modelo: apenas 55 (NF-e) e 65 (NFC-e)
	if mod != "55" && mod != "65" {
		return fmt.Errorf("modelo %s não suportado (aceito: 55, 65)", mod)
	}

	// Validação XML-04: apenas saídas (tpNF=1) — rejeita entradas com mensagem clara
	if strings.TrimSpace(inf.Ide.TpNF) != "1" {
		return fmt.Errorf("XML não é uma NF-e de saída (tpNF=%s) — este validador só aceita XMLs de saída", inf.Ide.TpNF)
	}

	chave := extractChave(proc)
	if len(chave) != 44 {
		return fmt.Errorf("chave de acesso inválida ou ausente")
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	nfeID, err := insertNFeSaidaHeader(tx, companyID, chave, inf)
	if err != nil {
		return fmt.Errorf("erro ao persistir nota: %w", err)
	}

	if len(inf.Det) > 0 {
		if err := insertNFeItens(tx, nfeID, companyID, inf.Det); err != nil {
			log.Printf("[XMLUpload] itens error [%s]: %v", chave, err)
			// Não abortar: falha em itens não invalida a nota principal
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao confirmar transação: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// XMLUploadHandler — POST /api/xml/upload
// Aceita um ou mais .xml, ou um .zip com múltiplos XMLs de NF-e de saída.
// Segurança: T-02-01 (anti-ZIP-bomb), T-02-02 (path traversal),
//            T-02-04 (escopo por company_id via JWT), T-02-05 (auth obrigatória).
// ---------------------------------------------------------------------------

func XMLUploadHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			jsonErr(w, http.StatusMethodNotAllowed, "Método não permitido")
			return
		}

		// T-02-05: autenticação via JWT — nunca aceitar company_id do corpo/query
		companyID, err := erpBridgeGetCompany(db, r)
		if err != nil {
			jsonErr(w, http.StatusUnauthorized, "Não autenticado")
			return
		}

		// T-02-01: validar Content-Length ANTES de ler o body
		if r.ContentLength > MaxUploadFileBytes {
			jsonErr(w, http.StatusRequestEntityTooLarge, "Arquivo excede limite de 5GB")
			return
		}

		// Parsear multipart mantendo ≤64MB em RAM; arquivos maiores vão para disco
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			jsonErr(w, http.StatusRequestEntityTooLarge, "Arquivo excede limite de 5GB")
			return
		}

		fileHeaders := r.MultipartForm.File["file"]
		if len(fileHeaders) == 0 {
			jsonErr(w, http.StatusBadRequest, "Campo 'file' não encontrado no formulário")
			return
		}

		var xmlFiles []namedXML

		for _, fhdr := range fileHeaders {
			fh, err := fhdr.Open()
			if err != nil {
				jsonErr(w, http.StatusInternalServerError, "Erro ao ler arquivo: "+err.Error())
				return
			}

			ext := strings.ToLower(filepath.Ext(fhdr.Filename))
			if ext == ".zip" {
				// Gravar arquivo em disco temporário para não carregar arquivos grandes na RAM
				tmpFile, tmpErr := os.CreateTemp("", "xmlupload-*.zip")
				if tmpErr != nil {
					fh.Close()
					jsonErr(w, http.StatusInternalServerError, "Erro ao criar arquivo temporário")
					return
				}
				tmpPath := tmpFile.Name()
				written, copyErr := io.Copy(tmpFile, io.LimitReader(fh, MaxUploadFileBytes+1))
				tmpFile.Close()
				fh.Close()
				if copyErr != nil {
					os.Remove(tmpPath)
					jsonErr(w, http.StatusInternalServerError, "Erro ao ler arquivo: "+copyErr.Error())
					return
				}
				if written > MaxUploadFileBytes {
					os.Remove(tmpPath)
					jsonErr(w, http.StatusRequestEntityTooLarge, "Arquivo excede limite de 5GB")
					return
				}
				extracted, archErr := extractXMLsFromZipFile(tmpPath)
				os.Remove(tmpPath)
				if archErr != nil {
					log.Printf("[XMLUpload] company=%s arquivo=%s erro: %v", companyID, fhdr.Filename, archErr)
					jsonErr(w, http.StatusBadRequest, "Erro ao processar '"+fhdr.Filename+"': "+archErr.Error())
					return
				}
				log.Printf("[XMLUpload] company=%s arquivo=%s extraídos: %d XMLs", companyID, fhdr.Filename, len(extracted))
				xmlFiles = append(xmlFiles, extracted...)
			} else if ext == ".xml" {
				rawData, readErr := io.ReadAll(io.LimitReader(fh, MaxSingleXMLBytes+1))
				fh.Close()
				if readErr != nil {
					jsonErr(w, http.StatusInternalServerError, "Erro ao ler arquivo: "+readErr.Error())
					return
				}
				if int64(len(rawData)) > MaxSingleXMLBytes {
					jsonErr(w, http.StatusRequestEntityTooLarge, "XML excede limite de 10MB por arquivo")
					return
				}
				xmlFiles = append(xmlFiles, namedXML{Name: fhdr.Filename, Data: rawData})
			} else {
				fh.Close()
				log.Printf("[XMLUpload] company=%s arquivo=%s formato não suportado (ext=%s)", companyID, fhdr.Filename, ext)
				jsonErr(w, http.StatusBadRequest, "Formato não suportado: envie .xml ou .zip")
				return
			}
		}

		if len(xmlFiles) == 0 {
			jsonErr(w, http.StatusBadRequest, "Nenhum arquivo XML válido encontrado. Verifique se os arquivos têm extensão .xml.")
			return
		}

		log.Printf("[XMLUpload] company=%s total=%d arquivo(s)", companyID, len(xmlFiles))

		result := xmlUploadResult{Erros: []xmlUploadError{}}
		for _, xf := range xmlFiles {
			if err := processSingleXML(db, companyID, xf); err != nil {
				log.Printf("[XMLUpload] company=%s file=%s err=%v", companyID, xf.Name, err)
				result.Rejeitados++
				result.Erros = append(result.Erros, xmlUploadError{Arquivo: xf.Name, Erro: err.Error()})
			} else {
				result.Importados++
			}
		}
		result.Total = len(xmlFiles)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}
